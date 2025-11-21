package metrics

// MetricSink: pluggable sink interface
type MetricSink interface {
	IncCounter(name string, labels map[string]string)
	SetGauge(name string, labels map[string]string, value float64)
}

// MetricHub: dispatches metric updates to registered sinks
type MetricHub struct {
	sinks []MetricSink
}

func NewMetricHub() *MetricHub {
	return &MetricHub{sinks: []MetricSink{}}
}

// adds a new sink to the hub
func (h *MetricHub) RegisterSink(sink MetricSink) {
	h.sinks = append(h.sinks, sink)
}

// invokes each sink to increment counter metric
func (h *MetricHub) IncCounter(name string, labels map[string]string) {
	for _, sink := range h.sinks {
		sink.IncCounter(name, labels)
	}
}

// invokes each sink to set gauge metric
func (h *MetricHub) SetGauge(name string, labels map[string]string, value float64) {
	for _, sink := range h.sinks {
		sink.SetGauge(name, labels, value)
	}
}
