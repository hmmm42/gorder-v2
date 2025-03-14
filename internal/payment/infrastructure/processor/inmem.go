package processor

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/entity"
)

type InmemProcessor struct{}

func NewInmemProcessor() *InmemProcessor {
	return &InmemProcessor{}
}

func (i InmemProcessor) CreatePaymentLink(ctx context.Context, order *entity.Order) (string, error) {
	return "inmem-payment-link", nil
}
