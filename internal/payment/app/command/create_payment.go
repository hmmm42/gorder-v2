package command

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/convertor"
	"github.com/hmmm42/gorder-v2/common/decorator"
	"github.com/hmmm42/gorder-v2/common/entity"
	"github.com/hmmm42/gorder-v2/common/logging"
	"github.com/hmmm42/gorder-v2/payment/domain"
	"github.com/sirupsen/logrus"
)

type CreatePayment struct {
	Order *entity.Order
}

type CreatePaymentHandler decorator.CommandHandler[CreatePayment, string]

type createPaymentHandler struct {
	processor domain.Processor
	orderGRPC OrderService
}

func NewCreatePaymentHandler(
	processor domain.Processor,
	orderGRPC OrderService,
	logger *logrus.Entry,
	metricClient decorator.MetricsClient,
) CreatePaymentHandler {
	return decorator.ApplyCommandDecorators[CreatePayment, string](
		createPaymentHandler{processor: processor, orderGRPC: orderGRPC},
		logger,
		metricClient,
	)
}

func (c createPaymentHandler) Handle(ctx context.Context, cmd CreatePayment) (string, error) {
	var err error
	defer logging.WhenCommandExecute(ctx, "CreatePaymentHandler", cmd, err)
	link, err := c.processor.CreatePaymentLink(ctx, cmd.Order)
	if err != nil {
		return "", err
	}
	newOrder := &entity.Order{
		ID:          cmd.Order.ID,
		CustomerID:  cmd.Order.CustomerID,
		Status:      "waiting_for_payment",
		Items:       cmd.Order.Items,
		PaymentLink: link,
	}
	err = c.orderGRPC.UpdateOrder(ctx, convertor.NewOrderConvertor().EntityToProto(newOrder))
	return link, err
}
