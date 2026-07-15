package handlers

import (
	"context"
	"log/slog"

	"github.com/abibby/salusa/request"
	jackett "github.com/webtor-io/go-jackett"
)

type TorrentSearchRequest struct {
	Query string `query:"q"`

	Ctx     context.Context `inject:""`
	Jackett *jackett.Client `inject:""`
	Log     *slog.Logger    `inject:""`
}

var TorrentSearch = request.Handler(func(r *TorrentSearchRequest) ([]jackett.Result, error) {
	r.Log.Info("test", "query", r.Query)
	results, err := r.Jackett.Fetch(r.Ctx,
		jackett.NewRawSearch().WithQuery(r.Query).Build(),
	)
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []jackett.Result{}
	}
	return results, nil
})
