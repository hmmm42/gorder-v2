package query

import (
	"context"
	"fmt"
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
	redisLockPrefix      = "check_stock_"
	redisSemaphorePrefix = "check_item_semaphore_"
	maxConcurrency       = 50                     // 每个商品的最大并发数
	maxRetries           = 3                      // 最大重试次数
	baseRetryDelay       = 100 * time.Millisecond // 基础重试延迟
)

type CheckIfItemsInStock struct {
	Items []*entity.ItemWithQuantity
}

type CheckIfItemsInStockHandler decorator.QueryHandler[CheckIfItemsInStock, []*entity.Item]

type checkIfItemsInStockHandler struct {
	stockRepo domain.Repository
	//stripeAPI *integration.StripeAPI
	stripeAPI integration.StripeProductFetcher
}

func NewCheckIfItemsInStockHandler(
	stockRepo domain.Repository,
	stripeAPI integration.StripeProductFetcher,
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

	// 使用重试机制处理信号量获取
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避重试延迟
			delay := time.Duration(attempt) * baseRetryDelay
			logging.Infof(ctx, logrus.Fields{
				"attempt": attempt,
				"delay":   delay,
			}, "retrying after semaphore acquisition failure")

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// 1. 尝试获取所有商品的信号量
		err = h.acquireSemaphores(ctx, query.Items)
		if err != nil {
			// 如果是并发限制错误且还有重试次数，继续重试
			if h.isConcurrencyLimitError(err) && attempt < maxRetries {
				continue
			}
			// 其他错误或达到最大重试次数，直接返回
			return nil, err
		}

		// 获取信号量成功，继续执行后续逻辑
		defer h.releaseSemaphores(ctx, query.Items)

		// 2. 获取商品信息
		res, err = h.getProductInfo(ctx, query.Items)
		if err != nil {
			return nil, err
		}

		// 3. 检查并扣减库存
		if err = h.checkStock(ctx, query.Items); err != nil {
			return nil, err
		}

		return res, nil
	}

	// 理论上不会到达这里，但为了完整性
	return nil, errors.New("max retries exceeded")
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

// ConcurrencyLimitError 并发限制错误类型
type ConcurrencyLimitError struct {
	ItemID  string
	Current int64
	Max     int64
}

func (e ConcurrencyLimitError) Error() string {
	return fmt.Sprintf("exceed max concurrency for item %s: current=%d, max=%d",
		e.ItemID, e.Current, e.Max)
}

// acquireSemaphores 获取所有商品的信号量
func (h checkIfItemsInStockHandler) acquireSemaphores(ctx context.Context, items []*entity.ItemWithQuantity) error {
	var acquiredItems []*entity.ItemWithQuantity

	for _, item := range items {
		cnt, err := redis.Incr(ctx, redis.LocalClient(), redisSemaphorePrefix+item.ID)
		if err != nil {
			// 如果获取失败，释放已获取的信号量
			h.releaseSemaphoresForItems(ctx, acquiredItems)
			return errors.Wrapf(err, "acquire semaphore error: itemID=%s", item.ID)
		}

		if cnt > maxConcurrency {
			// 超过并发限制，释放当前信号量并返回特定错误类型
			redis.Decr(ctx, redis.LocalClient(), redisSemaphorePrefix+item.ID)
			h.releaseSemaphoresForItems(ctx, acquiredItems)
			return ConcurrencyLimitError{
				ItemID:  item.ID,
				Current: cnt,
				Max:     maxConcurrency,
			}
		}

		// 设置过期时间，防止死锁
		_, _ = redis.Expire(ctx, redis.LocalClient(), redisSemaphorePrefix+item.ID, 30*time.Second)
		acquiredItems = append(acquiredItems, item)
	}
	return nil
}

// releaseSemaphores 释放所有商品的信号量
func (h checkIfItemsInStockHandler) releaseSemaphores(ctx context.Context, items []*entity.ItemWithQuantity) {
	for _, item := range items {
		redis.Decr(ctx, redis.LocalClient(), redisSemaphorePrefix+item.ID)
	}
}

// releaseSemaphoresForItems 释放指定商品的信号量
func (h checkIfItemsInStockHandler) releaseSemaphoresForItems(ctx context.Context, items []*entity.ItemWithQuantity) {
	for _, item := range items {
		redis.Decr(ctx, redis.LocalClient(), redisSemaphorePrefix+item.ID)
	}
}

// getProductInfo 获取商品信息
func (h checkIfItemsInStockHandler) getProductInfo(ctx context.Context, items []*entity.ItemWithQuantity) ([]*entity.Item, error) {
	var res []*entity.Item
	for _, item := range items {
		p, err := h.stripeAPI.GetProductByID(ctx, item.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "get product info failed for item %s", item.ID)
		}
		res = append(res, entity.NewItem(item.ID, p.Name, item.Quantity, p.DefaultPrice.ID))
	}
	return res, nil
}

// isConcurrencyLimitError 判断是否为并发限制错误
func (h checkIfItemsInStockHandler) isConcurrencyLimitError(err error) bool {
	var concurrencyErr ConcurrencyLimitError
	return errors.As(err, &concurrencyErr)
}

//func getStubPriceID(id string) string {
//	priceID, ok := stub[id]
//	if !ok {
//		priceID = stub["1"]
//	}
//	return priceID
//}
