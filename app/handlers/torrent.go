package handlers

import (
	"context"

	"github.com/abibby/page/config"

	// "github.com/abibby/page/services/qbittorrent"
	"github.com/abibby/salusa/request"
	qbittorrent "github.com/autobrr/go-qbittorrent"
	jackett "github.com/webtor-io/go-jackett"
)

type TorrentSearchRequest struct {
	Query string `query:"q"`

	Ctx     context.Context `inject:""`
	Jackett *jackett.Client `inject:""`
}

var TorrentSearch = request.Handler(func(r *TorrentSearchRequest) ([]jackett.Result, error) {
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

type TorrentActiveRequest struct {
	Query string `query:"q"`

	Ctx    context.Context     `inject:""`
	Client *qbittorrent.Client `inject:""`
	Cfg    *config.Config      `inject:""`
}

var TorrentActive = request.Handler(func(r *TorrentActiveRequest) ([]qbittorrent.Torrent, error) {
	return r.Client.GetTorrentsCtx(r.Ctx, qbittorrent.TorrentFilterOptions{
		Tag: r.Cfg.QbitTag,
	})
})

type TorrentAddRequest struct {
	URL string `json:"url"`

	Ctx    context.Context     `inject:""`
	Client *qbittorrent.Client `inject:""`
	Cfg    *config.Config      `inject:""`
}

var TorrentAdd = request.Handler(func(r *TorrentAddRequest) (any, error) {
	return r.Client.AddTorrentFromUrlCtx(r.Ctx, r.URL, map[string]string{
		"tags": r.Cfg.QbitTag,
	})
})
