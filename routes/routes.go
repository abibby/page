package routes

import (
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

	r.Handle("/d", handlers.CalibreData)
	r.Group("/api", func(r *router.Router) {
		r.Get("/list", handlers.BookList)
		r.Get("/book/{id}", handlers.BookView)
		r.Post("/book", handlers.BookImport)

		r.Get("/torrent/search", handlers.TorrentSearch)
		r.Get("/hardcover/search", handlers.HardcoverSearch)
	})
	r.Handle("/", fileserver.WithFallback(ui.Content, "dist", "index.html", nil))
}
