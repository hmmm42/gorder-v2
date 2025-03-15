package command

import (
	"context"
	"fmt"

	"github.com/hmmm42/gorder-v2/common/broker"
	"github.com/hmmm42/gorder-v2/common/convertor"
	"github.com/hmmm42/gorder-v2/common/decorator"
	"github.com/hmmm42/gorder-v2/common/entity"
	"github.com/hmmm42/gorder-v2/common/logging"
	"github.com/hmmm42/gorder-v2/order/app/query"
	domain "github.com/hmmm42/gorder-v2/order/domain/order"
	"github.com/hmmm42/gorder-v2/order/domain/service"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/status"
)

type CreateOrder struct {
	CustomerID string
	Items      []*entity.ItemWithQuantity
}

type CreateOrderResult struct {
	OrderID string
}

type CreateOrderHandler decorator.CommandHandler[CreateOrder, *CreateOrderResult]

type createOrderHandler struct {
	orderRepo      domain.Repository
	stockGRPC      query.StockService
	eventPublisher domain.EventPublisher
}

func NewCreateOrderHandler(
	orderRepo domain.Repository,
	stockGRPC query.StockService,
	eventPublisher domain.EventPublisher,
	logger *logrus.Logger,
	metricClient decorator.MetricsClient,
) CreateOrderHandler {
	if orderRepo == nil {
		panic("orderRepo is nil")
	}
	if stockGRPC == nil {
		panic("stockGRPC is nil")
	}
	if eventPublisher == nil {
		panic("eventPublisher is nil")
	}
	return decorator.ApplyCommandDecorators[CreateOrder, *CreateOrderResult](
		createOrderHandler{
			orderRepo:      orderRepo,
			stockGRPC:      stockGRPC,
			eventPublisher: eventPublisher,
		},
		logger,
		metricClient,
	)
}

func (c createOrderHandler) Handle(ctx context.Context, cmd CreateOrder) (*CreateOrderResult, error) {
	var err error
	defer logging.WhenCommandExecute(ctx, "CreateOrderHandler", cmd, err)

	t := otel.Tracer("rabbitmq")
	ctx, span := t.Start(ctx, fmt.Sprintf("rabbitmq.%s.publish", broker.EventOrderCreated))
	defer span.End()

	validItems, err := c.validate(ctx, cmd.Items)
	if err != nil {
		return nil, err
	}

	pendingOrder, err := domain.NewPendingOrder(cmd.CustomerID, validItems)
	if err != nil {
		return nil, err
	}
	o, err := service.NewOrderDomainService(c.orderRepo, c.eventPublisher).CreateOrder(ctx, *pendingOrder)
	return &CreateOrderResult{OrderID: o.ID}, nil
}

func (c createOrderHandler) validate(ctx context.Context, items []*entity.ItemWithQuantity) ([]*entity.Item, error) {
	if len(items) == 0 {
		return nil, errors.New("must have at least one item")
	}
	items = packItems(items)
	resp, err := c.stockGRPC.CheckIfItemsInStock(ctx, convertor.NewItemWithQuantityConvertor().EntitiesToProtos(items))
	if err != nil {
		return nil, status.Convert(err).Err()
	}
	return convertor.NewItemConvertor().ProtosToEntities(resp.Items), nil
}

func packItems(items []*entity.ItemWithQuantity) []*entity.ItemWithQuantity {
	merged := make(map[string]int32)
	for _, item := range items {
		merged[item.ID] += item.Quantity
	}
	var res []*entity.ItemWithQuantity
	for id, quantity := range merged {
		res = append(res, entity.NewItemWithQuantity(id, quantity))
	}
	return res
}
