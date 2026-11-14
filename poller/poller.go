package poller

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Tata-Matata/aria-vsphere-metrics-collector/logger"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/metrics"
)

// defines how data received from polled source will be processed
// into metrics by concrete poller and pushed to MetricHub
type MetricProcessor interface {
	ProcessAndPushMetrics(data []byte, hub *metrics.MetricHub) error
	Name() string
}

// generic poller gets json data from a REST API endpoint,
// processes it and sends metric to MetricHub.
type Poller struct {
	Name       string
	URL        string
	Interval   time.Duration
	Hub        *metrics.MetricHub
	HttpClient *http.Client

	// strategy pattern - custom logic to extract metric from response and push to hub
	Processor MetricProcessor
}

func New(processor MetricProcessor, url string, interval time.Duration, hub *metrics.MetricHub) *Poller {
	pollerName := processor.Name()
	logger.Info("Creating %s to poll URL %s every %d seconds", pollerName, url, interval)

	return &Poller{
		Processor: processor,
		Name:      pollerName,
		URL:       url,
		Interval:  interval,
		Hub:       hub,
		HttpClient: &http.Client{
			Timeout: POLL_TIMEOUT_SEC * time.Second,
		},
	}
}

// poll at regular intervals
func (poller *Poller) Start(context context.Context) {
	ticker := time.NewTicker(poller.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := poller.pollOnce(context); err != nil {
				log.Printf("poller %s error: %v", poller.Name, err)
			}
		//for graceful shutdown
		// since poller runs in background goroutine
		case <-context.Done():
			log.Printf("poller %s stopping", poller.Name)
			return
		}
	}
}

// perform single poll operation
func (poller *Poller) pollOnce(context context.Context) error {

	//make HTTP GET request
	req, err := http.NewRequestWithContext(context, "GET", poller.URL, nil)
	if err != nil {
		return logger.Error("poller %s failed to create GET request for polling: %v", poller.Name, err)
	}

	//execute request
	resp, err := poller.HttpClient.Do(req)
	if err != nil {
		return logger.Error("poller's %s polling request failed: %v", poller.Name, err)
	}
	defer resp.Body.Close()

	//check response status
	if resp.StatusCode != http.StatusOK {
		return logger.Error("Request from poller %s returned non-200 HTTP code %d", poller.Name, resp.StatusCode)
	}

	//read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return logger.Error("poller %s failed to read response body: %v", poller.Name, err)
	}

	//process and push metrics
	err = poller.Processor.ProcessAndPushMetrics(body, poller.Hub)
	if err != nil {
		return logger.Error("Poller %s failed to process and push metric to hub: %v", poller.Name, err)
	}

	return nil
}
