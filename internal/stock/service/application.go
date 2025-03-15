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
	"github.com/spf13/viper"
)

func NewApplication(_ context.Context) app.Application {
	db := persistent.NewMySQL()
	stockRepo := adapters.NewMySQLStockRepository(db)
	logger := logrus.StandardLogger()
	stripeAPI := integration.NewStripeAPI()
	metricClient := metrics.NewPrometheusMetricsClient(&metrics.PrometheusMetricsClientConfig{
		Host:        viper.GetString("stock.metrics_export_addr"),
		ServiceName: viper.GetString("stock.service-name"),
	})
	return app.Application{
		Commands: app.Commands{},
		Queries: app.Queries{
			CheckIfItemsInStock: query.NewCheckIfItemsInStockHandler(stockRepo, stripeAPI, logger, metricClient),
			GetItems:            query.NewGetItemsHandler(stockRepo, logger, metricClient),
		},
	}
}
