package app

import (
	"context"

	"github.com/abibby/page/app/events"
	"github.com/abibby/page/app/jobs"
	"github.com/abibby/page/app/models"
	"github.com/abibby/page/app/providers"
	"github.com/abibby/page/config"
	"github.com/abibby/page/migrations"
	"github.com/abibby/page/resources"
	"github.com/abibby/page/routes"
	"github.com/abibby/salusa/event"
	"github.com/abibby/salusa/event/cron"
	"github.com/abibby/salusa/kernel"
	"github.com/abibby/salusa/openapidoc"
	"github.com/abibby/salusa/salusadi"
	"github.com/abibby/salusa/view"
	"github.com/go-openapi/spec"
	"github.com/google/uuid"
)

var Kernel = kernel.New(
	kernel.Config(config.Load),
	kernel.Bootstrap(
		salusadi.Register[*models.User](migrations.Use()),
		view.Register(resources.Content, "**/*.html"),
		providers.Register,
		func(ctx context.Context) error {
			openapidoc.RegisterFormat[uuid.UUID]("uuid")
			return nil
		},
	),
	kernel.Services(
		cron.Service().
			Schedule("*/5 * * * *", &events.ImportEvent{}),
		event.Service(
			event.NewListener[*jobs.ImportJob](),
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
