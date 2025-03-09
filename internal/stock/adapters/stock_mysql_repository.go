package adapters

import (
	"context"

	"github.com/hmmm42/gorder-v2/stock/entity"
	"github.com/hmmm42/gorder-v2/stock/infrastructure/persistent"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type MySQLStockRepository struct {
	db *persistent.MySQL
}

func (m MySQLStockRepository) UpdateStock(
	ctx context.Context,
	data []*entity.ItemWithQuantity,
	updateFn func(
		ctx context.Context,
		existing []*entity.ItemWithQuantity,
		query []*entity.ItemWithQuantity,
	) ([]*entity.ItemWithQuantity, error),
) error {
	return m.db.StartTransaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if err != nil {
				logrus.Warnf("update stock transaction err=%v", err)

			}
		}()
		var dest []*persistent.StockModel
		// TODO: change table name
		if err = tx.Table("o_stock").Where("product_id IN ?", getIDFromEntities(data)).Find(&dest).Error; err != nil {
			return err
		}
		existing := m.unmarshalFromDatabase(dest)

		updated, err := updateFn(ctx, existing, data)
		if err != nil {
			return err
		}

		for _, upd := range updated {
			if err = tx.Table("o_stock").Where("product_id = ?", upd.ID).Update("quantity", upd.Quantity).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func getIDFromEntities(items []*entity.ItemWithQuantity) []string {
	var ids []string
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func NewMySQLStockRepository(db *persistent.MySQL) *MySQLStockRepository {
	return &MySQLStockRepository{db: db}
}

func (m MySQLStockRepository) GetItems(ctx context.Context, ids []string) ([]*entity.Item, error) {
	//TODO implement me
	panic("implement me")
}

func (m MySQLStockRepository) GetStock(ctx context.Context, ids []string) ([]*entity.ItemWithQuantity, error) {
	data, err := m.db.BatchGetStockByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	var result []*entity.ItemWithQuantity
	for _, d := range data {
		result = append(result, &entity.ItemWithQuantity{
			ID:       d.ProductID,
			Quantity: d.Quantity,
		})
	}
	return result, nil
}

func (m MySQLStockRepository) unmarshalFromDatabase(dest []*persistent.StockModel) []*entity.ItemWithQuantity {
	var result []*entity.ItemWithQuantity
	for _, d := range dest {
		result = append(result, &entity.ItemWithQuantity{
			ID:       d.ProductID,
			Quantity: d.Quantity,
		})
	}
	return result

}
