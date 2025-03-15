package query

import (
	"context"
	"strings"
	"time"

	"github.com/hmmm42/gorder-v2/common/decorator"
	"github.com/hmmm42/gorder-v2/common/entity"
	"github.com/hmmm42/gorder-v2/common/handler/redis"
	"github.com/hmmm42/gorder-v2/common/logging"
	domain "github.com/hmmm42/gorder-v2/stock/domain/stock"
	"github.com/hmmm42/gorder-v2/stock/infrastructure/integration"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	redisLockPrefix = "check_stock_"
)

type CheckIfItemsInStock struct {
	Items []*entity.ItemWithQuantity
}

type CheckIfItemsInStockHandler decorator.QueryHandler[CheckIfItemsInStock, []*entity.Item]

type checkIfItemsInStockHandler struct {
	stockRepo domain.Repository
	stripeAPI *integration.StripeAPI
}

func NewCheckIfItemsInStockHandler(
	stockRepo domain.Repository,
	stripeAPI *integration.StripeAPI,
	logger *logrus.Logger,
	metricClient decorator.MetricsClient,
) CheckIfItemsInStockHandler {
	if stockRepo == nil {
		panic("nil stockRepo")
	}
	if stripeAPI == nil {
		panic("nil stripeAPI")
	}
	return decorator.ApplyQueryDecorators[CheckIfItemsInStock, []*entity.Item](
		checkIfItemsInStockHandler{
			stockRepo: stockRepo,
			stripeAPI: stripeAPI,
		},
		logger,
		metricClient,
	)
}

// Deprecated
var stub = map[string]string{
	"1": "price_1QyUx5GHy2qP0Z7nrtA1HnvP",
	"2": "price_1QyoB0GHy2qP0Z7n6ZcfsoQs",
}

func (h checkIfItemsInStockHandler) Handle(ctx context.Context, query CheckIfItemsInStock) (res []*entity.Item, err error) {
	if err = lock(ctx, getLockKey(query)); err != nil {
		return nil, errors.Wrapf(err, "redis lock error: key=%s", getLockKey(query))
	}
	defer func() {
		if err = unlock(ctx, getLockKey(query)); err != nil {
			logging.Warnf(ctx, nil, "redis unlock fail, err=%v", err)
		}
	}()

	defer func() {
		f := logrus.Fields{
			"query": query,
			"res":   res,
		}
		if err != nil {
			logging.Errorf(ctx, f, "checkIfItemsInStock err=%v", err)
		} else {
			logging.Infof(ctx, f, "%s", "checkIfItemsInStock success")
		}
	}()

	for _, item := range query.Items {
		p, err := h.stripeAPI.GetProductByID(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		res = append(res, entity.NewItem(item.ID, p.Name, item.Quantity, p.DefaultPrice.ID))
	}
	if err = h.checkStock(ctx, query.Items); err != nil {
		return nil, err
	}
	return res, nil
}

func getLockKey(query CheckIfItemsInStock) string {
	var ids []string
	for _, i := range query.Items {
		ids = append(ids, i.ID)
	}
	return redisLockPrefix + strings.Join(ids, "_")
}

func lock(ctx context.Context, key string) error {
	return redis.SetNX(ctx, redis.LocalClient(), key, "1", 5*time.Minute)
}

func unlock(ctx context.Context, key string) error {
	return redis.Del(ctx, redis.LocalClient(), key)
}

func (h checkIfItemsInStockHandler) checkStock(ctx context.Context, query []*entity.ItemWithQuantity) error {
	var ids []string
	for _, item := range query {
		ids = append(ids, item.ID)
	}
	records, err := h.stockRepo.GetStock(ctx, ids)
	if err != nil {
		return err
	}
	idQuantityMap := make(map[string]int32)
	for _, r := range records {
		idQuantityMap[r.ID] += r.Quantity
	}
	var (
		ok       = true
		failedOn []struct {
			ID   string
			Want int32
			Have int32
		}
	)
	for _, item := range query {
		if item.Quantity > idQuantityMap[item.ID] {
			ok = false
			failedOn = append(failedOn, struct {
				ID   string
				Want int32
				Have int32
			}{ID: item.ID, Want: item.Quantity, Have: idQuantityMap[item.ID]})
		}
	}

	if ok {
		return h.stockRepo.UpdateStock(ctx, query, func(
			ctx context.Context,
			existing []*entity.ItemWithQuantity,
			query []*entity.ItemWithQuantity,
		) ([]*entity.ItemWithQuantity, error) {
			var newItems []*entity.ItemWithQuantity
			for _, e := range existing {
				for _, q := range query {
					if e.ID == q.ID {
						iq, err := entity.NewValidItemWithQuantity(e.ID, e.Quantity-q.Quantity)
						if err != nil {
							return nil, err
						}
						newItems = append(newItems, iq)
					}
				}
			}
			return newItems, nil
		})
	}
	return domain.ExceedStockError{FailedOn: failedOn}
}

//func getStubPriceID(id string) string {
//	priceID, ok := stub[id]
//	if !ok {
//		priceID = stub["1"]
//	}
//	return priceID
//}
