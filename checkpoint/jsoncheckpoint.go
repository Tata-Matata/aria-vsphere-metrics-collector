package checkpoint

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/Tata-Matata/aria-vsphere-metrics-collector/util"
)

type JSONCheckpoint struct {
	lock sync.Mutex

	FilePath string

	// storage maps for checkpointing and json serialization: metric -> (labelKey -> value)
	// deploy_total{errType="unathenticated", status="failure"}
	// counterValues["deploy_total"]["errType=unathenticated|status=failure"] = 42
	// we have to store these redundantly because Prometheus client does not provide
	// public API to extract all current label-value pairs and numeric values.
	CounterValues map[string]map[string]float64
	GaugeValues   map[string]map[string]float64
}

// creates a new JSON checkpoint with empty maps.
func NewJSONCheckpoint(filePath string) *JSONCheckpoint {
	return &JSONCheckpoint{
		FilePath:      filePath,
		CounterValues: make(map[string]map[string]float64),
		GaugeValues:   make(map[string]map[string]float64),
	}
}

func (checkpoint *JSONCheckpoint) IncCounter(name string, labels map[string]string) {
	checkpoint.lock.Lock()
	defer checkpoint.lock.Unlock()

	if _, exists := checkpoint.CounterValues[name]; !exists {
		checkpoint.CounterValues[name] = map[string]float64{}
	}

	// merge metric labels into single string key
	//"errType=unathenticated|status=failure"
	key := util.JoinMapEntries(labels, KEY_VAL_SEPARATOR)

	checkpoint.CounterValues[name][key]++
}

func (checkpoint *JSONCheckpoint) SetGauge(name string, labels map[string]string, value float64) {
	checkpoint.lock.Lock()
	defer checkpoint.lock.Unlock()

	if _, exists := checkpoint.GaugeValues[name]; !exists {
		checkpoint.GaugeValues[name] = map[string]float64{}
	}

	// merge metric labels into single string key
	//"errType=unathenticated|status=failure"
	key := util.JoinMapEntries(labels, KEY_VAL_SEPARATOR)

	checkpoint.GaugeValues[name][key] = value
}

// Save writes the current metric maps to the JSON file
func (jc *JSONCheckpoint) Save() error {
	jc.lock.Lock()
	defer jc.lock.Unlock()

	file, err := os.Create(jc.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(struct {
		Counters map[string]map[string]float64 `json:"counters"`
		Gauges   map[string]map[string]float64 `json:"gauges"`
	}{
		Counters: jc.CounterValues,
		Gauges:   jc.GaugeValues,
	})
}

// Load restores metric maps from the JSON file
func (jc *JSONCheckpoint) Load() error {
	jc.lock.Lock()
	defer jc.lock.Unlock()

	file, err := os.Open(jc.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		Counters map[string]map[string]float64 `json:"counters"`
		Gauges   map[string]map[string]float64 `json:"gauges"`
	}{}

	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return err
	}

	jc.CounterValues = data.Counters
	jc.GaugeValues = data.Gauges
	return nil
}

// StartPeriodic starts a goroutine that periodically saves metrics to the file
func (jc *JSONCheckpoint) StartPeriodic(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := jc.Save(); err != nil {
				// log error in real implementation
				// fmt.Println("Checkpoint save error:", err)
			}
		}
	}()
}
