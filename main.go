package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Tata-Matata/aria-vsphere-metrics-collector/handlers"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/logger"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/metrics"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/poller"
	"github.com/Tata-Matata/aria-vsphere-metrics-collector/prometheus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger

	logger, err := logger.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	fmt.Printf("Writing logs to %v\n", logger.Dir)
	defer logger.Close()

	// Create metric hub and prometheus sink
	hub := metrics.NewMetricHub()
	//TODO: make checkpoint file and interval configurable
	promSink := prometheus.NewSink(METRICS_BACKUP_FILE, METRICS_BACKUP_INTERVAL_SEC*time.Second)
	hub.RegisterSink(promSink)

	// set global handler hub
	handlers.Hub = hub

	// Example pollers: replace with your real URLs and labels
	// These poll remote GET endpoints periodically and set gauges
	examplePollers := []*poller.Poller{
		poller.NewPoller("http://localhost:5000/gauge1", "external_gauge_1", map[string]string{"source": "fake-api"}, 15*time.Second, hub),
		poller.NewPoller("http://localhost:5000/gauge2", "external_gauge_2", map[string]string{"source": "fake-api"}, 20*time.Second, hub),
	}
	for _, p := range examplePollers {
		p.Start()
	}

	// HTTP routes for receiving pushed events and for Prometheus scraping
	http.HandleFunc("/event", handlers.EventHandler) // legacy format
	http.HandleFunc("/push", handlers.PushHandler)   // generic push
	http.HandleFunc("/health", handlers.HealthHandler)
	http.Handle("/metrics", promhttp.Handler())

	addr := ":8080"
	fmt.Println("Starting exporter on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
