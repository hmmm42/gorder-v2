package ports

import (
	"context"
	"github.com/hmmm42/gorder-v2/common/genproto/stockpb"
	"github.com/hmmm42/gorder-v2/stock/app"
)

type GRPCServer struct {
	app app.Application
}

func NewGRPCServer(app app.Application) *GRPCServer {
	return &GRPCServer{app: app}
}

func (G GRPCServer) GetItems(ctx context.Context, request *stockpb.GetItemsRequest) (*stockpb.GetItemsResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (G GRPCServer) CheckIfItemInStock(ctx context.Context, request *stockpb.CheckIfItemInStockRequest) (*stockpb.CheckIfItemInStockResponse, error) {
	//TODO implement me
	panic("implement me")
}
