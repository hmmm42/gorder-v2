package processor

import (
	"context"
	"github.com/hmmm42/gorder-v2/common/genproto/orderpb"
)

type InmemProcessor struct{}

func NewInmemProcessor() *InmemProcessor {
	return &InmemProcessor{}
}

func (i InmemProcessor) CreatePaymentLink(_ context.Context, _ *orderpb.Order) (string, error) {
	return "inmem-payment-link", nil
}
