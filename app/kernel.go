package app

import (
	"context"

	"github.com/abibby/page/app/events"
	"github.com/abibby/page/app/jobs"
	"github.com/abibby/page/app/providers"
	"github.com/abibby/page/config"
	"github.com/abibby/page/migrations"
	"github.com/abibby/page/resources"
	"github.com/abibby/page/routes"
	"github.com/go-openapi/spec"
	"github.com/google/uuid"
	"gosalusa.com/clog"
	"gosalusa.com/database"
	"gosalusa.com/event"
	"gosalusa.com/event/cron"
	"gosalusa.com/kernel"
	"gosalusa.com/openapidoc"
	"gosalusa.com/pubsub/channelpubsub"
	"gosalusa.com/request"
	"gosalusa.com/view"
	"gosalusa.com/wfs"
)

var Kernel = kernel.New(
	kernel.Config(config.Load),
	kernel.Bootstrap(
		view.Register(resources.Content, "**/*.html"),
		providers.Register,
		func(ctx context.Context) error {
			openapidoc.RegisterFormat[uuid.UUID]("uuid")
			return nil
		},
		kernel.Register(func(ctx context.Context, c *config.Config) {
			database.Register(ctx, c.Database, migrations.Use())
			channelpubsub.Register(ctx)

			clog.RegisterDefault(ctx)
			request.Register(ctx)
			event.Register(ctx)
			wfs.RegisterLocal(ctx, c.LibraryPath)
		}),
	),
	kernel.Services(
		cron.Service().
			Schedule("*/5 * * * *", &events.ScanEvent{}).
			Schedule("* * * * *", &events.ImportEvent{}),
		event.Service(
			event.NewListener[*jobs.ImportJob](),
			event.NewListener[*jobs.ScanJob](),
		),
	),
	kernel.InitRoutes(routes.InitRoutes),
	kernel.APIDocumentation(
		openapidoc.Info(spec.InfoProps{
			Title:       "Page API",
			Description: `This is the API documentaion for page`,
		}),
		openapidoc.BasePath("/api"),
	),
)
