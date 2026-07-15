package handlers

import (
	"net/http"

	"github.com/abibby/page/config"
	"github.com/abibby/salusa/request"
)

type CalibreDataRequest struct {
	Cfg *config.Config `inject:""`
}

var CalibreData = request.Handler(func(r *CalibreDataRequest) (http.Handler, error) {
	return http.FileServer(http.Dir(r.Cfg.CalibreLibrary)), nil
})
