package query

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/decorator"
	domain "github.com/hmmm42/gorder-v2/stock/domain/stock"
	"github.com/hmmm42/gorder-v2/stock/entity"
	"github.com/sirupsen/logrus"
)

type GetItems struct {
	ItemsIDs []string
}

type GetItemsHandler decorator.QueryHandler[GetItems, []*entity.Item]

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
	return decorator.ApplyQueryDecorators[GetItems, []*entity.Item](
		getItemsHandler{stockRepo: stockRepo},
		logger,
		metricClient,
	)
}

func (g getItemsHandler) Handle(ctx context.Context, query GetItems) ([]*entity.Item, error) {
	items, err := g.stockRepo.GetItems(ctx, query.ItemsIDs)
	if err != nil {
		return nil, err
	}
	return items, nil
}
