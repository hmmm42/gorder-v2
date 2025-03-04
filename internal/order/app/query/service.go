package query

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/genproto/orderpb"
	"github.com/hmmm42/gorder-v2/common/genproto/stockpb"
)

type StockService interface {
	CheckIfItemsInStock(ctx context.Context, item []*orderpb.ItemWithQuantity) (*stockpb.CheckIfItemInStockResponse, error)
	GetItems(ctx context.Context, itemIDs []string) ([]*orderpb.Item, error)
}
