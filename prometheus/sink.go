package prometheus

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// --------------------
// PrometheusSink
// --------------------
//
// This sink registers CounterVec/GaugeVec and also keeps simple maps
// of numeric values so snapshotting / checkpointing is straightforward.
type PrometheusSink struct {
	// protected by lock
	lock sync.Mutex

	// prometheus metric vectors
	counters map[string]*prometheus.CounterVec
	gauges   map[string]*prometheus.GaugeVec

	// label name templates for each metric (sorted)
	labelNames map[string][]string

	// storage maps for checkpointing: metric -> (labelKey -> value)
	counterValues map[string]map[string]float64
	gaugeValues   map[string]map[string]float64
}

func NewSink() *PrometheusSink {
	return &PrometheusSink{
		counters:      make(map[string]*prometheus.CounterVec),
		gauges:        make(map[string]*prometheus.GaugeVec),
		labelNames:    make(map[string][]string),
		counterValues: make(map[string]map[string]float64),
		gaugeValues:   make(map[string]map[string]float64),
	}
}

func (p *PrometheusSink) getOrCreateCounter(name string, labelNames []string) *prometheus.CounterVec {
	p.lock.Lock()
	defer p.lock.Unlock()
	if c, ok := p.counters[name]; ok {
		return c
	}
	c := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: name + " counter",
	}, labelNames)
	p.counters[name] = c
	p.labelNames[name] = labelNames
	p.counterValues[name] = make(map[string]float64)
	prometheus.MustRegister(c)
	return c
}

func (p *PrometheusSink) getOrCreateGauge(name string, labelNames []string) *prometheus.GaugeVec {
	p.lock.Lock()
	defer p.lock.Unlock()
	if g, ok := p.gauges[name]; ok {
		return g
	}
	g := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: name + " gauge",
	}, labelNames)
	p.gauges[name] = g
	p.labelNames[name] = labelNames
	p.gaugeValues[name] = make(map[string]float64)
	prometheus.MustRegister(g)
	return g
}

// IncCounter implements MetricSink
func (p *PrometheusSink) IncCounter(name string, labels map[string]string) {
	labelNames := keysFromLabels(labels)
	counter := p.getOrCreateCounter(name, labelNames)

	// update prometheus metric
	counter.With(labels).Add(1)

	// store in our map for checkpointing
	key := labelKeyFromLabels(labelNames, labels)
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.counterValues[name]; !ok {
		p.counterValues[name] = make(map[string]float64)
	}
	p.counterValues[name][key] += 1
}

// SetGauge implements MetricSink
func (p *PrometheusSink) SetGauge(name string, labels map[string]string, value float64) {
	labelNames := keysFromLabels(labels)
	gauge := p.getOrCreateGauge(name, labelNames)

	gauge.With(labels).Set(value)

	// store
	key := labelKeyFromLabels(labelNames, labels)
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.gaugeValues[name]; !ok {
		p.gaugeValues[name] = make(map[string]float64)
	}
	p.gaugeValues[name][key] = value
}

func keysFromLabels(labels map[string]string) []string {
	if labels == nil {
		return []string{}
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func labelKeyFromLabels(keys []string, labels map[string]string) string {
	// produce stable key like "k1=v1|k2=v2"
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, "|")
}

type metricSnapshot struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
	Type   string            `json:"type"` // "counter" or "gauge"
}

// SaveCheckpoint writes current counters/gauges to filename as JSON
func (p *PrometheusSink) SaveCheckpoint(filename string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	var out []metricSnapshot

	// counters
	for name, m := range p.counterValues {
		labelNames := p.labelNames[name]
		for key, v := range m {
			labels := map[string]string{}
			if key != "" {
				// reconstruct labels map from key and labelNames
				parts := strings.Split(key, "|")
				for _, part := range parts {
					kv := strings.SplitN(part, "=", 2)
					if len(kv) == 2 {
						labels[kv[0]] = kv[1]
					}
				}
			}
			out = append(out, metricSnapshot{
				Name:   name,
				Labels: labels,
				Value:  v,
				Type:   "counter",
			})
		}
		_ = m
		_ = labelNames
	}

	// gauges
	for name, m := range p.gaugeValues {
		for key, v := range m {
			labels := map[string]string{}
			if key != "" {
				parts := strings.Split(key, "|")
				for _, part := range parts {
					kv := strings.SplitN(part, "=", 2)
					if len(kv) == 2 {
						labels[kv[0]] = kv[1]
					}
				}
			}
			out = append(out, metricSnapshot{
				Name:   name,
				Labels: labels,
				Value:  v,
				Type:   "gauge",
			})
		}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	tmp := filename + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filename)
}

// LoadCheckpoint loads saved metrics and applies them to the sink
func (p *PrometheusSink) LoadCheckpoint(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// nothing to load
		return nil
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	var in []metricSnapshot
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}

	// apply
	for _, sink := range in {
		if sink.Type == "counter" {
			// counters are cumulative, so Add the stored value
			p.IncCounter(sink.Name, sink.Labels)
		} else if sink.Type == "gauge" {
			p.SetGauge(sink.Name, sink.Labels, sink.Value)
		}
	}
	return nil
}

// StartAutoSave spawns goroutine saving every interval
func (p *PrometheusSink) StartAutoSave(filename string, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			if err := p.SaveCheckpoint(filename); err != nil {
				// minimal logging — avoid importing log package here
				println("SaveCheckpoint error:", err.Error())
			}
		}
	}()
}
