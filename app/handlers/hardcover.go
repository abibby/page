package handlers

import (
	"context"

	"github.com/abibby/page/services/hardcover"
	"github.com/abibby/salusa/request"
)

type HardcoverSearchRequest struct {
	Query string `query:"q"`

	Ctx context.Context   `inject:""`
	HC  *hardcover.Client `inject:""`
}

var HardcoverSearch = request.Handler(func(r *HardcoverSearchRequest) (any, error) {
	return r.HC.Query(r.Ctx, r.Query)
})
