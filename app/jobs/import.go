package jobs

import (
	"context"
	"log/slog"

	"github.com/abibby/page/app/events"
	"github.com/abibby/page/services/importer"
)

type ImportJob struct {
	Importer *importer.Importer `inject:""`
	Log      *slog.Logger       `inject:""`
}

func (l *ImportJob) Handle(ctx context.Context, e *events.ImportEvent) error {
	l.Log.Info("starting import pass")
	return l.Importer.RunPass(ctx)
}
