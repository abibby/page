package jobs

import (
	"context"
	"log/slog"

	"github.com/abibby/page/app/events"
	"github.com/abibby/page/services/library"
)

type ScanJob struct {
	Library *library.Store `inject:""`
	Log     *slog.Logger   `inject:""`
}

func (j *ScanJob) Handle(ctx context.Context, e *events.ScanEvent) error {
	j.Log.Info("starting library scan")
	return j.Library.Scan(ctx)
}
