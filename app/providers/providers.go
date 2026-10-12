package providers

import (
	"context"
	"log/slog"

	"github.com/abibby/page/app/models"
	"github.com/abibby/page/config"
	"github.com/abibby/page/services/hardcover"
	"github.com/abibby/page/services/importer"
	"github.com/abibby/page/services/library"
	"github.com/autobrr/go-qbittorrent"
	"github.com/jmoiron/sqlx"
	"github.com/webtor-io/go-jackett"
	"gosalusa.com/database/model/modeldi"
	"gosalusa.com/di"
)

var registrar = []func(context.Context){
	modeldi.Register[*models.Book],
}

func Add(register func(context.Context)) {
	registrar = append(registrar, register)
}

// Register registers any custom di providers
func Register(ctx context.Context) error {
	for _, register := range registrar {
		register(ctx)
	}

	di.RegisterLazySingletonWith(ctx, func(cfg *config.Config) (*qbittorrent.Client, error) {
		return qbittorrent.NewClient(qbittorrent.Config{
			Host:     cfg.QbitURL,
			Username: cfg.QbitUsername,
			Password: cfg.QbitPassword,
		}), nil
	})

	type libraryWith struct {
		Ctx context.Context `inject:""`
		Cfg *config.Config  `inject:""`
		DB  *sqlx.DB        `inject:""`
	}
	di.RegisterLazySingletonWith(ctx, func(w *libraryWith) (*library.Store, error) {
		return library.New(w.Ctx, w.Cfg, w.DB)
	})

	type appWith struct {
		Cfg      *config.Config      `inject:""`
		HC       *hardcover.Client   `inject:""`
		Importer *library.Store      `inject:""`
		Qbt      *qbittorrent.Client `inject:""`
		Log      *slog.Logger        `inject:""`
	}
	di.RegisterLazySingletonWith(ctx, func(w *appWith) (*importer.Importer, error) {
		return importer.New(w.Cfg, w.HC, w.Importer, w.Qbt, w.Log), nil
	})

	di.RegisterLazySingletonWith(ctx, func(cfg *config.Config) (*hardcover.Client, error) {
		return hardcover.New(cfg.HardcoverURL, cfg.HardcoverToken), nil
	})
	di.RegisterLazySingletonWith(ctx, func(cfg *config.Config) (*jackett.Client, error) {
		return jackett.New(jackett.Settings{
			ApiURL: cfg.JackettUrl,
			ApiKey: cfg.JackettApiKey,
		})
	})

	return nil
}
