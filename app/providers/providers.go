package providers

import (
	"context"
	"log/slog"

	"github.com/abibby/page/config"
	"github.com/abibby/page/services/calibre"
	"github.com/abibby/page/services/calibredb"
	"github.com/abibby/page/services/hardcover"
	"github.com/abibby/page/services/importer"
	"github.com/abibby/salusa/di"
	"github.com/autobrr/go-qbittorrent"
	"github.com/webtor-io/go-jackett"
)

// var ModelRegistrar = modeldi.NewModelRegistrar()
var registrar = []func(context.Context){}

func Add(register func(context.Context)) {
	registrar = append(registrar, register)
}

// Register registers any custom di providers
func Register(ctx context.Context) error {
	for _, register := range registrar {
		register(ctx)
	}

	di.RegisterLazySingletonWith(ctx, func(cfg *config.Config) (*calibredb.Client, error) {
		return calibredb.NewClient(cfg.CalibredbBin, &calibredb.GlobalFlags{
			LibraryPath: cfg.CalibreLibrary,
		}), nil
	})
	di.RegisterLazySingletonWith(ctx, func(cfg *config.Config) (*qbittorrent.Client, error) {
		return qbittorrent.NewClient(qbittorrent.Config{
			Host:     cfg.QbitURL,
			Username: cfg.QbitUsername,
			Password: cfg.QbitPassword,
		}), nil
	})

	type calibreWith struct {
		Cfg             *config.Config    `inject:""`
		CalibredbClient *calibredb.Client `inject:""`
	}
	di.RegisterLazySingletonWith(ctx, func(w *calibreWith) (*calibre.Importer, error) {
		return calibre.NewClient(w.Cfg, w.CalibredbClient), nil
	})

	type appWith struct {
		Cfg      *config.Config      `inject:""`
		HC       *hardcover.Client   `inject:""`
		Importer *calibre.Importer   `inject:""`
		Qbt      *qbittorrent.Client `inject:""`
		Log      *slog.Logger        `inject:""`
	}
	di.RegisterLazySingletonWith(ctx, func(w *appWith) (*importer.Importer, error) {
		return importer.New(w.Cfg, w.HC, w.Importer, w.Qbt, w.Log), nil
	})

	di.RegisterLazySingletonWith(ctx, func(cfg *config.Config) (*calibredb.Client, error) {
		return calibredb.NewClient(cfg.CalibredbBin, &calibredb.GlobalFlags{
			LibraryPath: cfg.CalibreLibrary,
		}), nil
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
