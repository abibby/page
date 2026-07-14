package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/abibby/fileserver"
	"github.com/abibby/page/app"
	"github.com/abibby/page/internal/calibre"
	"github.com/abibby/page/internal/calibredb"
	"github.com/abibby/page/internal/config"
	"github.com/abibby/page/internal/hardcover"
	"github.com/abibby/page/server/handlers"
	"github.com/abibby/page/server/ui"
	"github.com/abibby/salusa/di"
	"github.com/abibby/salusa/request"
	"github.com/abibby/salusa/router"
	"github.com/abibby/salusa/salusaconfig"
	"github.com/webtor-io/go-jackett"
)

func main() {
	cfg, err := config.Load("./.env")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	dp := di.NewDependencyProvider()
	ctx := di.ContextWithDependencyProvider(context.Background(), dp)

	di.RegisterSingleton(ctx, func() *config.Config {
		return cfg
	})

	di.RegisterSingleton(ctx, func() *calibredb.Client {
		return calibredb.NewClient(cfg.CalibredbBin, &calibredb.GlobalFlags{
			LibraryPath: cfg.CalibreLibrary,
		})
	})

	type importerWith struct {
		Cfg             *config.Config    `inject:""`
		CalibredbClient *calibredb.Client `inject:""`
	}
	di.RegisterLazySingletonWith(ctx, func(w *importerWith) (*calibre.Importer, error) {
		return calibre.NewClient(w.Cfg, w.CalibredbClient), nil
	})

	type appWith struct {
		Cfg      *config.Config    `inject:""`
		HC       *hardcover.Client `inject:""`
		Importer *calibre.Importer `inject:""`
	}
	di.RegisterLazySingletonWith(ctx, func(w *appWith) (*app.App, error) {
		return app.New(w.Cfg, w.HC, w.Importer), nil
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

	di.RegisterSingleton(ctx, func() salusaconfig.Config {
		return cfg
	})

	r := router.New()

	r.Register(ctx)
	_ = request.Register(ctx)

	r.Handle("/d", http.StripPrefix("/d", http.FileServer(http.Dir(cfg.CalibreLibrary))))
	r.Group("/api", func(r *router.Router) {
		r.Get("/list", handlers.BookList)
		r.Get("/book/{id}", handlers.BookView)
		r.Post("/book", handlers.BookImport)

		r.Get("/torrent/search", handlers.TorrentSearch)
		r.Get("/hardcover/search", handlers.HardcoverSearch)
	})
	r.Handle("/", fileserver.WithFallback(ui.Content, "dist", "index.html", nil))

	err = r.Validate(ctx)
	if err != nil {
		log.Fatalf("router validation failed: %v", err)
	}
	err = dp.Validate(ctx)
	if err != nil {
		log.Fatalf("di validation failed: %v", err)
	}

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.GetHTTPPort()),
		Handler: r,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	err = s.ListenAndServe()
	if err != nil {
		slog.Error("http server failed", "error", err)
		os.Exit(1)
	}
}
