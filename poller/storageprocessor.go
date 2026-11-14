package poller

import (
	"encoding/json"

	"github.com/Tata-Matata/aria-vsphere-metrics-collector/logger"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/metrics"
)

type StorageProcessor struct {
}

func (sp *StorageProcessor) Name() string {
	return "StoragePoller"
}

func (sp *StorageProcessor) ProcessAndPushMetrics(data []byte, hub *metrics.MetricHub) error {
	var st struct {
		Capacity   int64  `json:"capacity_bytes"`
		Used       int64  `json:"used_bytes"`
		Node       string `json:"node"`
		Datacenter string `json:"datacenter"`
	}

	if err := json.Unmarshal(data, &st); err != nil {
		return logger.Error("StorageProcessor failed to unmarshal data: %v", err)
	}

	labels := map[string]string{"datacenter": st.Datacenter}

	// push metrics directly to hub
	// storage_free_bytes{datacenter="dc1"} 123456789
	hub.SetGauge("storage_free_bytes", labels, float64(st.Capacity))

	return nil
}
