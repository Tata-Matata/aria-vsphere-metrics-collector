# Aria vSphere Metrics Collector

A Go-based metrics collection and aggregation service that bridges vSphere and Aria (VMware Aria) metrics with Prometheus monitoring. This project collects metrics from remote endpoints, normalizes them through pluggable processors, and exposes them in Prometheus format for monitoring and observability.

## Overview

This metrics collector operates as a dual-mode service:
- **Active Polling**: Periodically fetches data from remote REST API endpoints (e.g., vSphere storage, Aria deployments)
- **Passive Reception**: Accepts pushed metrics from external clients via HTTP endpoints
- **Metric Export**: Exposes aggregated metrics in Prometheus format for scraping

The architecture uses a pluggable processor pattern and a metric hub design for flexible metric handling and extensibility.

## Features

- **Generic Polling Framework**: Strategy-pattern based pollers that can handle different data sources
- **Metric Hub Architecture**: Decoupled metric processing with support for multiple sinks (Prometheus, checkpoints, etc.)
- **Prometheus Integration**: Native Prometheus client library for metrics collection and HTTP exposure
- **Checkpoint/Persistence**: Automatic metric persistence to JSON checkpoints for recovery
- **Flexible HTTP Endpoints**: Support for both legacy (`/event`) and generic (`/push`) metric ingestion formats
- **Health Checks**: Built-in health check endpoint for monitoring service availability
- **Structured Logging**: Dedicated logger with file-based logging output

## Architecture

### Core Components

#### MetricHub (`metrics/`)
The central dispatch mechanism for metric updates. It manages multiple metric sinks and broadcasts metric changes (counters and gauges) to all registered sinks.

```go
type MetricSink interface {
    IncCounter(name string, labels map[string]string)
    SetGauge(name string, labels map[string]string, value float64)
}
```

#### Poller (`poller/`)
Generic polling framework that fetches data from remote endpoints at regular intervals. Uses the Strategy pattern with pluggable `MetricProcessor` implementations:

- **StorageProcessor**: Extracts and processes vSphere storage metrics
- **DeployProcessor**: Processes Aria deployment metrics

#### Prometheus Sink (`prometheus/`)
Integrates with the Prometheus Go client library to expose metrics in the Prometheus format. Automatically persists metrics to a JSON checkpoint file for recovery.

#### Handlers (`handlers/`)
HTTP request handlers for metric ingestion:
- `/event` - Legacy metric event format
- `/push` - Generic metric push endpoint
- `/metrics` - Prometheus metrics scraping endpoint
- `/health` - Health check endpoint

#### Logger (`logger/`)
Structured logging with file-based output. Logs are written to a configurable directory.

#### Checkpoint (`checkpoint/`)
Persistence layer for metrics using JSON format. Enables recovery of metrics across restarts.

## Getting Started

### Prerequisites

- Go 1.23.0 or later
- No external runtime dependencies (all dependencies managed via go.mod)

### Installation

```bash
git clone https://github.com/Tata-Matata/aria-vsphere-metrics-collector.git
cd aria-vsphere-metrics-collector
go build
```

### Running

```bash
./aria-vsphere-metrics-collector
```

The service will start on `http://localhost:8080` by default.

## Configuration

Currently, configuration is hardcoded in `main.go`. Key parameters:

| Parameter | Value | Description |
|-----------|-------|-------------|
| `DEFAULT_LISTEN_ADDR` | `:8080` | HTTP server listen address |
| `METRICS_BACKUP_FILE` | `metrics_checkpoint.json` | Checkpoint file for metric persistence |
| `METRICS_BACKUP_INTERVAL_SEC` | `60` | Interval (seconds) for persisting metrics to checkpoint |

**Note**: Configuration is a high-priority TODO item for future releases.

### Pollers Configuration

Pollers are currently hardcoded in `main.go`:
- Storage poller: polls `https://vSPHEREURL:80/storage` every 15 seconds
- Deploy poller: polls `https://ARIAURL:80/deployments` every 20 seconds

## API Endpoints

### Metric Ingestion

#### POST `/event` (Legacy)
Legacy format for pushing events/metrics.

```bash
curl -X POST http://localhost:8080/event \
  -H "Content-Type: application/json" \
  -d '{...}'
```

#### POST `/push` (Generic)
Generic endpoint for pushing metrics.

```bash
curl -X POST http://localhost:8080/push \
  -H "Content-Type: application/json" \
  -d '{...}'
```

### Metrics Export

#### GET `/metrics`
Prometheus format metrics scraping endpoint.

```bash
curl http://localhost:8080/metrics
```

### Health Check

#### GET `/health`
Service health check.

```bash
curl http://localhost:8080/health
```

## Project Structure

```
aria-vsphere-metrics-collector/
├── main.go                 # Application entry point
├── constants.go            # Global constants
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── checkpoint/             # Metric persistence layer
│   ├── checkpoint.go       # Checkpoint interface
│   └── jsoncheckpoint.go   # JSON implementation
├── handlers/               # HTTP request handlers
│   └── handlers.go
├── logger/                 # Logging utilities
│   ├── log.go
│   └── constants.go
├── metrics/                # Metric hub and interfaces
│   └── metrics.go
├── poller/                 # Polling framework and processors
│   ├── poller.go           # Generic poller
│   ├── constants.go
│   ├── storageprocessor.go # vSphere storage metrics
│   └── deployprocessor.go  # Aria deployment metrics
├── prometheus/             # Prometheus sink implementation
│   └── sink.go
└── util/                   # Utility functions
    ├── util.go
    └── constants.go
```

## Development

### Building

```bash
go build
```

### Running Tests

```bash
go test ./...
```

### Dependencies

- **prometheus/client_golang**: Prometheus instrumentation library and HTTP client

All dependencies are defined in `go.mod` and managed via standard Go module tooling.

## Testing with Fake ARIA API

For local development and integration testing without requiring a real ARIA instance, this project can be used with the `fake-aria-api mock` server. The `fake-aria-api` provides mock endpoints for deployments and datastores, allowing you to test the metrics collector's polling and event handling logic in isolation. 
To use it, configure the poller endpoints in `main.go` to point to `http://localhost:10000` (the fake-aria-api default port) instead of production ARIA or vSphere URLs, then run both the `fake-aria-api server` and the metrics collector. This approach enables rapid iteration, automated testing, and CI/CD integration without external dependencies.

## Design Patterns

### Strategy Pattern (MetricProcessor)
Different data sources are handled by implementing the `MetricProcessor` interface:

```go
type MetricProcessor interface {
    ProcessAndPushMetrics(data []byte, hub *metrics.MetricHub) error
    Name() string
}
```

### Hub/Spoke Pattern (MetricHub)
Centralized dispatcher for metrics allows decoupling of metric production and consumption:
- Multiple sources can push metrics to a single hub
- Multiple sinks can subscribe to metric updates

### Sink Pattern (MetricSink)
Pluggable sinks allow flexible output:
- Prometheus sink for monitoring
- JSON checkpoint sink for persistence
- Easy to add additional sinks (e.g., time-series database, logging, etc.)



## Known Limitations & Future Work

See [TODO.md](TODO.md) for outstanding work items:

### High Priority
- Make all parameters in main configurable (ports, endpoints, intervals, pollers)
- Add authentication/authorization
- Support dynamic poller configuration

### Medium Priority
- Generic checkpoint abstraction for use by all sinks
- Unified metric data model for checkpoint and sink interaction

### Low Priority
- Additional metric sinks
- Enhanced monitoring and observability

## Logging

Logs are written to a directory created by the logger during initialization. The directory path is printed to stdout on startup

## Graceful Shutdown

The application uses Go contexts for graceful shutdown of background goroutines (pollers). This allows in-flight polls to complete before termination.

