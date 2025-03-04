package ports

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/genproto/stockpb"
	"github.com/hmmm42/gorder-v2/stock/app"
	"github.com/hmmm42/gorder-v2/stock/app/query"
)

type GRPCServer struct {
	app app.Application
}

func NewGRPCServer(app app.Application) *GRPCServer {
	return &GRPCServer{app: app}
}

func (G GRPCServer) GetItems(ctx context.Context, request *stockpb.GetItemsRequest) (*stockpb.GetItemsResponse, error) {
	items, err := G.app.Queries.GetItems.Handle(ctx, query.GetItems{ItemsIDs: request.ItemIDs})
	if err != nil {
		return nil, err
	}
	return &stockpb.GetItemsResponse{Items: items}, nil
}

func (G GRPCServer) CheckIfItemInStock(ctx context.Context, request *stockpb.CheckIfItemInStockRequest) (*stockpb.CheckIfItemInStockResponse, error) {
	items, err := G.app.Queries.CheckIfItemsInStock.Handle(ctx, query.CheckIfItemsInStock{Items: request.Items})
	if err != nil {
		return nil, err
	}
	return &stockpb.CheckIfItemInStockResponse{
		InStock: 1,
		Items:   items,
	}, nil
}
