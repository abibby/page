package events

import (
	"github.com/abibby/salusa/event"
	"github.com/abibby/salusa/event/cron"
)

type ImportEvent struct {
	cron.CronEvent
}

var _ event.Event = (*ImportEvent)(nil)

func (e *ImportEvent) Type() event.EventType {
	return "page:import"
}
