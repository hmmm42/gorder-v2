package grpc

import (
	"context"
	"errors"

	"github.com/hmmm42/gorder-v2/common/genproto/orderpb"
	"github.com/hmmm42/gorder-v2/common/genproto/stockpb"
	"github.com/hmmm42/gorder-v2/common/logging"
)

type StockGRPC struct {
	client stockpb.StockServiceClient
}

func NewStockGRPC(client stockpb.StockServiceClient) *StockGRPC {
	return &StockGRPC{client: client}
}

func (s StockGRPC) CheckIfItemsInStock(ctx context.Context, items []*orderpb.ItemWithQuantity) (resp *stockpb.CheckIfItemInStockResponse, err error) {
	_, dLog := logging.WhenRequest(ctx, "StockGRPC.CheckIfItemsInStock", items)
	defer dLog(resp, &err)

	if items == nil {
		return nil, errors.New("grpc items cannot be nil")
	}
	return s.client.CheckIfItemInStock(ctx, &stockpb.CheckIfItemInStockRequest{Items: items})
}

func (s StockGRPC) GetItems(ctx context.Context, itemIDs []string) (items []*orderpb.Item, err error) {
	_, dLog := logging.WhenRequest(ctx, "StockGRPC.GetItems", items)
	defer dLog(items, &err)

	resp, err := s.client.GetItems(ctx, &stockpb.GetItemsRequest{ItemIDs: itemIDs})
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}
