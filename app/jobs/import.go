package jobs

import (
	"context"

	"github.com/abibby/page/app/events"
	"github.com/abibby/page/services/importer"
)

type ImportJob struct {
	Importer *importer.Importer `inject:""`
}

func (l *ImportJob) Handle(ctx context.Context, e *events.ImportEvent) error {
	l.Importer.RunPass(ctx)
	return nil
}
