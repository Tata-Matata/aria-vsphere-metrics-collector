package poller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Tata-Matata/aria-vsphere-metrics-collector/metrics"
)

// Simple poller that GETs a URL expecting JSON like {"value": 123.4}
// and sets a gauge metric in the MetricHub.

type Poller struct {
	URL        string
	MetricName string
	Labels     map[string]string
	Interval   time.Duration
	Hub        *metrics.MetricHub
	Client     *http.Client
}

func NewPoller(url, metric string, labels map[string]string, interval time.Duration, hub *metrics.MetricHub) *Poller {
	return &Poller{
		URL:        url,
		MetricName: metric,
		Labels:     labels,
		Interval:   interval,
		Hub:        hub,
		Client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (p *Poller) Start() {
	go func() {
		t := time.NewTicker(p.Interval)
		defer t.Stop()
		for range t.C {
			if err := p.pollOnce(); err != nil {
				fmt.Printf("Poller error (%s): %v\n", p.URL, err)
			}
		}
	}()
}

func (p *Poller) pollOnce() error {
	resp, err := p.Client.Get(p.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Expect either: {"value": number} or {"value": "123.4"} or a raw number
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		// If not object, try parse as plain number
		var val float64
		if err2 := json.Unmarshal(body, &val); err2 == nil {
			p.Hub.SetGauge(p.MetricName, p.Labels, val)
			return nil
		}
		return err
	}
	v, ok := parsed["value"]
	if !ok {
		return fmt.Errorf("no 'value' in response")
	}
	switch t := v.(type) {
	case float64:
		p.Hub.SetGauge(p.MetricName, p.Labels, t)
	case int:
		p.Hub.SetGauge(p.MetricName, p.Labels, float64(t))
	case string:
		// try parse numeric string
		var x float64
		if err := json.Unmarshal([]byte("\""+t+"\""), &x); err == nil {
			p.Hub.SetGauge(p.MetricName, p.Labels, x)
		} else {
			return fmt.Errorf("value is string and not numeric: %v", t)
		}
	default:
		return fmt.Errorf("unsupported value type %T", v)
	}
	return nil
}
