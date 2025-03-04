package grpc

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/genproto/orderpb"
	"github.com/hmmm42/gorder-v2/common/genproto/stockpb"
	"github.com/sirupsen/logrus"
)

type StockGRPC struct {
	client stockpb.StockServiceClient
}

func NewStockGRPC(client stockpb.StockServiceClient) *StockGRPC {
	return &StockGRPC{client: client}
}

func (s StockGRPC) CheckIfItemsInStock(ctx context.Context, item []*orderpb.ItemWithQuantity) (*stockpb.CheckIfItemInStockResponse, error) {
	resp, err := s.client.CheckIfItemInStock(ctx, &stockpb.CheckIfItemInStockRequest{Items: item})
	logrus.Info("stock_grpc response", resp)
	return resp, err
}

func (s StockGRPC) GetItems(ctx context.Context, itemIDs []string) ([]*orderpb.Item, error) {
	resp, err := s.client.GetItems(ctx, &stockpb.GetItemsRequest{ItemIDs: itemIDs})
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}
