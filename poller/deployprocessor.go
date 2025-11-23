package poller

import (
	"encoding/json"

	"github.com/Tata-Matata/aria-vsphere-metrics-collector/logger"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/metrics"
)

type DeployProcessor struct {
}

func (sp *DeployProcessor) Name() string {
	return "DeploymentsPoller"
}

// perform processing of data and push metrics to MetricHub
func (sp *DeployProcessor) ProcessAndPushMetrics(data []byte, hub *metrics.MetricHub) error {
	var st struct {
		ErrType string `json:"errtype"`
		Status  string `json:"datacenter"`
	}

	if err := json.Unmarshal(data, &st); err != nil {
		return logger.Error("DeployProcessor failed to unmarshal data: %v", err)
	}

	labels := map[string]string{"status": st.Status, "errtype": st.ErrType}

	// push metrics directly to hub
	hub.IncCounter("deploy_total", labels)
	return nil
}
