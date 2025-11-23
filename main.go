package main

import (
	"context"
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

	/*******************************************
	********** POLL clients for metrics ********
	*******************************************/

	// poll remote GET endpoints periodically
	//TODO: add docs why context is needed for graceful shutdown of background goroutines
	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	pollers := []*poller.Poller{
		poller.New(&poller.StorageProcessor{}, "https://vSPHEREURL:80/storage", 15*time.Second, hub),
		poller.New(&poller.DeployProcessor{}, "https://ARIAURL:80/deployments", 20*time.Second, hub),
	}
	for _, poller := range pollers {
		poller.Start(context)
	}

	/************************************************
	********** RECEIVE metrics pushed by clients*****
	*************************************************/
	// HTTP routes for receiving pushed events
	http.HandleFunc("/event", handlers.EventHandler) // legacy format
	http.HandleFunc("/push", handlers.PushHandler)   // generic push

	// for Prometheus scraping
	http.Handle("/metrics", promhttp.Handler())

	// health check endpoint
	http.HandleFunc("/health", handlers.HealthHandler)
	addr := DEFAULT_LISTEN_ADDR
	fmt.Println("Starting exporter on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
