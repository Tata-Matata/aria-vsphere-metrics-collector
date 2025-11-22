package checkpoint

import "time"

// Generic interface for backing up/loading metrics at intervals
type Checkpoint interface {
	Save() error
	Load() error
	StartPeriodic(interval time.Duration)
}
