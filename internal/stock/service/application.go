package service

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/metrics"
	"github.com/hmmm42/gorder-v2/stock/adapters"
	"github.com/hmmm42/gorder-v2/stock/app"
	"github.com/hmmm42/gorder-v2/stock/app/query"
	"github.com/hmmm42/gorder-v2/stock/infrastructure/integration"
	"github.com/hmmm42/gorder-v2/stock/infrastructure/persistent"
	"github.com/sirupsen/logrus"
)

func NewApplication(_ context.Context) app.Application {
	db := persistent.NewMySQL()
	stockRepo := adapters.NewMySQLStockRepository(db)
	logger := logrus.NewEntry(logrus.StandardLogger())
	stripeAPI := integration.NewStripeAPI()
	metricClient := metrics.TodoMetrics{}
	return app.Application{
		Commands: app.Commands{},
		Queries: app.Queries{
			CheckIfItemsInStock: query.NewCheckIfItemsInStockHandler(stockRepo, stripeAPI, logger, metricClient),
			GetItems:            query.NewGetItemsHandler(stockRepo, logger, metricClient),
		},
	}
}
