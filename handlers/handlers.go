package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Tata-Matata/aria-vsphere-metrics-collector/logger"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/metrics"
)

// This handlers package expects a global MetricHub instance set by main
var Hub *metrics.MetricHub

// Legacy event structure
type LegacyEvent struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
}

// Generic push structure for extensibility
type PushEvent struct {
	Name   string            `json:"name"`             // metric name
	Type   string            `json:"type"`             // "counter" or "gauge"
	Value  float64           `json:"value"`            // numeric value
	Labels map[string]string `json:"labels,omitempty"` // optional labels
}

// EventHandler handles legacy events like {"status":"success","errorType":""}
func EventHandler(w http.ResponseWriter, r *http.Request) {
	var e LegacyEvent
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &e); err != nil {
		http.Error(w, "invalid legacy event", http.StatusBadRequest)
		return
	}

	if e.Status == "" {
		http.Error(w, "missing status", http.StatusBadRequest)
		return
	}

	// increment events_total{status="<status>"} and optionally event_errors_total{type="<error>"}
	Hub.IncCounter("events_total", map[string]string{"status": e.Status})
	if e.ErrorType != "" {
		Hub.IncCounter("event_errors_total", map[string]string{"type": e.ErrorType})
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

// PushHandler handles generic pushes for counters/gauges
// POST JSON: {"name":"my_metric","type":"counter","value":1,"labels":{"a":"b"}}
func PushHandler(w http.ResponseWriter, r *http.Request) {
	var p PushEvent
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if p.Name == "" {
		http.Error(w, "missing metric name", http.StatusBadRequest)
		return
	}
	switch p.Type {
	case "counter":
		Hub.IncCounter(p.Name, p.Labels)
	case "gauge":
		Hub.SetGauge(p.Name, p.Labels, p.Value)
	default:
		http.Error(w, "unknown metric type (use 'counter' or 'gauge')", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

// Health check
func HealthHandler(respWriter http.ResponseWriter, request *http.Request) {
	respWriter.WriteHeader(http.StatusOK)
	logger.Info(fmt.Sprintf("Health check response %v %s", respWriter, "OK"))
}
