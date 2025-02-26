package ports

import (
	"context"
	"github.com/hmmm42/gorder-v2/common/genproto/stockpb"
)

type GRPCServer struct {
}

func NewGRPCServer() *GRPCServer {
	return &GRPCServer{}
}

func (G GRPCServer) GetItems(ctx context.Context, request *stockpb.GetItemsRequest) (*stockpb.GetItemsResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (G GRPCServer) CheckIfItemInStock(ctx context.Context, request *stockpb.CheckIfItemInStockRequest) (*stockpb.CheckIfItemInStockResponse, error) {
	//TODO implement me
	panic("implement me")
}
