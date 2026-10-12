package routes

import (
	"context"
	"net/http"

	"github.com/abibby/fileserver"
	"github.com/abibby/page/app/handlers"
	"github.com/abibby/page/ui"
	"gosalusa.com/auth"
	"gosalusa.com/clog"
	"gosalusa.com/request"
	"gosalusa.com/router"
)

func InitRoutes(r *router.Router) {
	r.Use(request.HandleErrors(func(ctx context.Context, err error) http.Handler {
		clog.Use(ctx).Error("request failed", "error", err)
		return nil
	}))
	r.Use(auth.AttachUser())

	r.Group("/api", func(r *router.Router) {
		r.Get("/book", handlers.BookList).Name("book.list")
		r.Get("/book/{id}", handlers.BookView).Name("book.view")
		r.Post("/book/import", handlers.BookImport).Name("book.import")
		r.Post("/book", handlers.BookAdd).Name("book.add")

		r.Get("/torrent/search", handlers.TorrentSearch).Name("torrent.search")
		r.Get("/torrent/active", handlers.TorrentActive).Name("torrent.active")
		r.Post("/torrent", handlers.TorrentAdd).Name("torrent.add")

		r.Get("/hardcover/search", handlers.HardcoverSearch).Name("hardcover.search")

		r.Handle("/d", http.StripPrefix("/api/d", handlers.LibraryData)).Name("data")
	})
	r.Handle("/", fileserver.WithFallback(ui.Content, "dist", "index.html", nil))
}
