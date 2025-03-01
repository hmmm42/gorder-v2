package service

import (
	"context"
	"github.com/hmmm42/gorder-v2/common/metrics"
	"github.com/hmmm42/gorder-v2/order/adapters"
	"github.com/hmmm42/gorder-v2/order/app"
	"github.com/hmmm42/gorder-v2/order/app/query"
	"github.com/sirupsen/logrus"
)

func NewApplication(ctx context.Context) app.Application {
	orderInmemRepo := adapters.NewMemoryOrderRepository()
	logger := logrus.NewEntry(logrus.StandardLogger())
	metricsClient := metrics.TodoMetrics{}
	return app.Application{
		Commands: app.Commands{},
		Queries: app.Queries{
			GetCustomerOrder: query.NewGetCustomerOrderHandler(orderInmemRepo, logger, metricsClient),
		},
	}
}
