package query

import (
	"context"
	"github.com/hmmm42/gorder-v2/common/decorator"
	"github.com/hmmm42/gorder-v2/common/genproto/orderpb"
	domain "github.com/hmmm42/gorder-v2/stock/domain/stock"
	"github.com/sirupsen/logrus"
)

type GetItems struct {
	ItemsIDs []string
}

type GetItemsHandler decorator.QueryHandler[GetItems, []*orderpb.Item]

type getItemsHandler struct {
	stockRepo domain.Repository
}

func NewGetItemsHandler(
	stockRepo domain.Repository,
	logger *logrus.Entry,
	metricClient decorator.MetricsClient,
) GetItemsHandler {
	if stockRepo == nil {
		panic("nil stockRepo")
	}
	return decorator.ApplyQueryDecorators[GetItems, []*orderpb.Item](
		getItemsHandler{stockRepo: stockRepo},
		logger,
		metricClient,
	)
}

func (g getItemsHandler) Handle(ctx context.Context, query GetItems) ([]*orderpb.Item, error) {
	items, err := g.stockRepo.GetItems(ctx, query.ItemsIDs)
	if err != nil {
		return nil, err
	}
	return items, nil
}
