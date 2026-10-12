package handlers

import (
	"net/http"

	"github.com/abibby/page/config"
	"gosalusa.com/request"
)

type LibraryDataRequest struct {
	Cfg *config.Config `inject:""`
}

var LibraryData = request.Handler(func(r *LibraryDataRequest) (http.Handler, error) {
	return http.FileServer(http.Dir(r.Cfg.LibraryPath)), nil
})
