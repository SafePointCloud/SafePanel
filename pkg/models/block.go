package models

import "time"

type BlockRecord struct {
	IP        string
	StartTime time.Time
	Duration  time.Duration
}
