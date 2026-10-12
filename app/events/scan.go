package events

import (
	"gosalusa.com/event"
	"gosalusa.com/event/cron"
)

type ScanEvent struct {
	cron.CronEvent
}

var _ event.Event = (*ScanEvent)(nil)

func (e *ScanEvent) Type() event.EventType {
	return "page:scan"
}
