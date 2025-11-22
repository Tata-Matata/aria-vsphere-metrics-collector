package checkpoint

import "time"

// Checkpoint is a generic interface for saving/restoring metrics.
type Checkpoint interface {
	Save() error
	Load() error
	StartPeriodic(interval time.Duration)
}
