package prometheus

import (
	"fmt"
	"sync"
	"time"

	"github.com/Tata-Matata/aria-vsphere-metrics-collector/checkpoint"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/logger"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/util"
	"github.com/prometheus/client_golang/prometheus"
)

// --------------------
// PrometheusSink
// --------------------
//
// This sink registers CounterVec/GaugeVec and also keeps simple maps
// of numeric values so snapshotting / checkpointing is straightforward.
type PrometheusSink struct {
	// protected by lock; prevents race conditions due to concurrent metric updates
	// from multiple goroutines (push events via POST, pull events via polling, Prometheus scrapes)
	lock sync.Mutex

	// metric name -> Prometheus defined metric vectors CounterVec/GaugeVec
	// counters["deploy_total"] = CounterVec(name="deploy_total", labels=["result"] // value: success | fail)
	// when we call sink.IncCounter("deploy_total", map[string]string{"result": "success"})
	// CounterVec is invoked: counters["deploy_total"].WithLabelValues("success").Inc()
	counters map[string]*prometheus.CounterVec
	gauges   map[string]*prometheus.GaugeVec

	// Prometheus requires label names to be known at metric registration time.
	// If we register metric deploy_total{errType="unathenticated", status="success"}
	// the map will contain (sorted) labelNames["deploy_total"] = []string{"errType", "status"}
	// Although labelNames are part of CounterVec/GaugeVec, we store them redundantly,  because they can't be retrieved later from vectors.
	// Prometheus intentionally hides the list of label names from CounterVec/GaugeVec
	labelNames map[string][]string

	// regularly backs up metric values to disk
	checkpoint *checkpoint.JSONCheckpoint
}

func NewSink(checkpointFile string, saveInterval time.Duration) *PrometheusSink {
	psink := &PrometheusSink{
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		labelNames: make(map[string][]string),
	}

	// Initialize checkpoint manager for regular backups
	if checkpointFile != "" {
		// create checkpoint
		psink.checkpoint = checkpoint.NewJSONCheckpoint(checkpointFile)

		// initialize maps inside checkpoint
		psink.checkpoint.CounterValues = make(map[string]map[string]float64)
		psink.checkpoint.GaugeValues = make(map[string]map[string]float64)

		// load previous metrics from  backup if exists into checkpoint maps
		if err := psink.checkpoint.Load(); err != nil {
			logger.Error(fmt.Sprint("Failed to load checkpoint:", err))
		} else {
			psink.restoreFromCheckpoint()
		}

		// start periodic backups
		psink.checkpoint.StartPeriodic(saveInterval)
	}

	return psink
}

// restores metric values from checkpoint into the sink
func (psink *PrometheusSink) restoreFromCheckpoint() {
	psink.lock.Lock()
	defer psink.lock.Unlock()

	checkpoint := psink.checkpoint

	// 1. Restore counters
	for metricName, series := range checkpoint.GetCounterValues() {
		vec := psink.getOrCreateCounter(metricName, psink.labelNames[metricName])
		for labelsKey, value := range series {
			//we stored labels joined by separator in a single string key,
			// need to deserialize back to map
			labels := util.MapFromString(labelsKey)
			vec.With(labels).Add(value)
		}
	}

	// 2. Restore gauges
	for name, series := range checkpoint.GetGaugeValues() {
		vec := psink.getOrCreateGauge(name, psink.labelNames[name])
		for labelsKey, value := range series {
			labels := util.MapFromString(labelsKey)
			vec.With(labels).Set(value)
		}
	}
}

// retrieves existing CounterVec or creates a new one if it doesn't exist
func (psink *PrometheusSink) getOrCreateCounter(name string, labelNames []string) *prometheus.CounterVec {

	// check if metric already exists
	if counterVec, ok := psink.counters[name]; ok {
		return counterVec
	}

	// create new CounterVec with specified label names
	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: name + " counter",
	}, labelNames)

	psink.counters[name] = counterVec
	psink.labelNames[name] = labelNames

	//tells Prometheus to track this metric and expose it on /metrics
	prometheus.MustRegister(counterVec)

	return counterVec
}

// retrieves existing GaugeVec or creates a new one if it doesn't exist
func (psink *PrometheusSink) getOrCreateGauge(name string, labelNames []string) *prometheus.GaugeVec {

	// check if metric already exists
	if gaugeVec, ok := psink.gauges[name]; ok {
		return gaugeVec
	}
	gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: name + " gauge",
	}, labelNames)
	psink.gauges[name] = gaugeVec
	psink.labelNames[name] = labelNames

	//tells Prometheus to track this metric and expose it on /metrics
	prometheus.MustRegister(gaugeVec)

	return gaugeVec
}

// increases counter metrics, implements MetricSink
func (psink *PrometheusSink) IncCounter(name string, labels map[string]string) {
	//prevent race conditions on concurrent access via multiple metric updates
	psink.lock.Lock()
	defer psink.lock.Unlock()

	labelNames := util.SortedKeysFromMap(labels)
	counter := psink.getOrCreateCounter(name, labelNames)

	// update Prometheus metric value
	counter.With(labels).Add(1)

	// update our internal map for backuping
	if psink.checkpoint != nil {
		psink.checkpoint.IncCounter(name, labels)
	}

}

// SetGauge implements MetricSink
func (psink *PrometheusSink) SetGauge(name string, labels map[string]string, value float64) {
	psink.lock.Lock()
	defer psink.lock.Unlock()

	labelNames := util.SortedKeysFromMap(labels)
	gauge := psink.getOrCreateGauge(name, labelNames)

	// update prometheus metric value
	gauge.With(labels).Set(value)

	/// update our internal map for backuping
	if psink.checkpoint != nil {
		psink.checkpoint.SetGauge(name, labels, value)
	}
}
