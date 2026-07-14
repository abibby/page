package handlers

import (
	"context"

	"github.com/abibby/salusa/request"
	jackett "github.com/webtor-io/go-jackett"
)

type TorrentSearchRequest struct {
	Query string `query:"q"`

	Ctx     context.Context `inject:""`
	Jackett *jackett.Client `inject:""`
}

var TorrentSearch = request.Handler(func(r *TorrentSearchRequest) (any, error) {
	return r.Jackett.Fetch(r.Ctx,
		jackett.NewRawSearch().WithQuery(r.Query).Build(),
	)
})
