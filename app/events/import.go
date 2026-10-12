package events

import (
	"gosalusa.com/event"
	"gosalusa.com/event/cron"
)

type ImportEvent struct {
	cron.CronEvent
}

var _ event.Event = (*ImportEvent)(nil)

func (e *ImportEvent) Type() event.EventType {
	return "page:import"
}
