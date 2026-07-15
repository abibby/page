package routes

import (
	"net/http"

	"github.com/abibby/fileserver"
	"github.com/abibby/page/app/handlers"
	"github.com/abibby/page/ui"
	"github.com/abibby/salusa/auth"
	"github.com/abibby/salusa/request"
	"github.com/abibby/salusa/router"
)

func InitRoutes(r *router.Router) {
	r.Use(request.HandleErrors())
	r.Use(auth.AttachUser())

	r.Group("/api", func(r *router.Router) {
		r.Get("/book", handlers.BookList).Name("book.list")
		r.Get("/book/{id}", handlers.BookView).Name("book.view")
		r.Post("/book", handlers.BookImport).Name("book.import")

		r.Get("/torrent/search", handlers.TorrentSearch).Name("torrent.search")
		r.Get("/hardcover/search", handlers.HardcoverSearch).Name("hardcover.search")

		r.Handle("/d", http.StripPrefix("/api/d", handlers.CalibreData)).Name("data")
	})
	r.Handle("/", fileserver.WithFallback(ui.Content, "dist", "index.html", nil))
}
