package adapters

import (
	"context"
	"github.com/hmmm42/gorder-v2/common/genproto/orderpb"
	domain "github.com/hmmm42/gorder-v2/stock/domain/stock"
	"sync"
)

type MemoryItemRepository struct {
	lock  *sync.RWMutex
	store map[string]*orderpb.Item
}

var stub = map[string]*orderpb.Item{
	"item_id": {
		ID:       "foo_item",
		Name:     "stub item",
		Quantity: 10000,
		PriceID:  "stub_item_price_id",
	},
}

func NewMemoryItemRepository() *MemoryItemRepository {
	return &MemoryItemRepository{
		lock:  &sync.RWMutex{},
		store: stub,
	}
}

func (m MemoryItemRepository) GetItems(ctx context.Context, ids []string) ([]*orderpb.Item, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	var (
		res     []*orderpb.Item
		missing []string
	)
	for _, id := range ids {
		if item, exists := m.store[id]; exists {
			res = append(res, item)
		} else {
			missing = append(missing, id)
		}
	}
	if len(res) == len(ids) {
		return res, nil
	}
	return nil, domain.NotFoundError{Missing: missing}
}
