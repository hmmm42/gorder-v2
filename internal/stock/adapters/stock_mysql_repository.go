package adapters

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/entity"
	"github.com/hmmm42/gorder-v2/stock/infrastructure/persistent"
	"github.com/hmmm42/gorder-v2/stock/infrastructure/persistent/builder"
	"github.com/pkg/errors"
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
		err = m.updatePessimistic(ctx, tx, data, updateFn)
		//err = m.updateOptimistic(ctx, tx, data, updateFn)
		return err
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
	query := builder.NewStock().ProductIDs(ids...)
	data, err := m.db.BatchGetStockByID(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "BatchGetStockByID error")
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

func (m MySQLStockRepository) unmarshalFromDatabase(dest []persistent.StockModel) []*entity.ItemWithQuantity {
	var result []*entity.ItemWithQuantity
	for _, d := range dest {
		result = append(result, entity.NewItemWithQuantity(d.ProductID, d.Quantity))
	}
	return result

}

func (m MySQLStockRepository) updatePessimistic(
	ctx context.Context,
	tx *gorm.DB,
	data []*entity.ItemWithQuantity,
	updateFn func(ctx context.Context, existing []*entity.ItemWithQuantity, query []*entity.ItemWithQuantity,
	) ([]*entity.ItemWithQuantity, error)) error {

	var dest []persistent.StockModel
	dest, err := m.db.BatchGetStockByID(ctx, builder.NewStock().ProductIDs(getIDFromEntities(data)...).ForUpdate())
	if err != nil {
		return errors.Wrap(err, "failed to find data")
	}

	existing := m.unmarshalFromDatabase(dest)
	updated, err := updateFn(ctx, existing, data)
	if err != nil {
		return err
	}

	// Create a map for quick lookup of query quantities by ID
	queryMap := make(map[string]int32)
	for _, query := range data {
		queryMap[query.ID] = query.Quantity
	}

	// Process each updated item
	for _, upd := range updated {
		if queryQty, exists := queryMap[upd.ID]; exists {
			if err = m.db.Update(ctx, tx, builder.NewStock().ProductIDs(upd.ID).QuantityGT(queryQty),
				map[string]any{
					//"quantity": upd.Quantity,
					"quantity": gorm.Expr("quantity - ?", queryQty),
				}); err != nil {
				return errors.Wrapf(err, "unable to update %s", upd.ID)
			}
		} else {
			return errors.Errorf("item %s not found in query data", upd.ID)
		}
	}
	return nil
}

func (m MySQLStockRepository) updateOptimistic(
	ctx context.Context,
	tx *gorm.DB,
	data []*entity.ItemWithQuantity,
	updateFn func(ctx context.Context, existing []*entity.ItemWithQuantity, query []*entity.ItemWithQuantity,
	) ([]*entity.ItemWithQuantity, error)) error {

	for _, queryData := range data {
		var newestRecord *persistent.StockModel
		newestRecord, err := m.db.GetStockByID(ctx, builder.NewStock().ProductIDs(queryData.ID))
		if err != nil {
			return err
		}
		if err = m.db.Update(
			ctx,
			tx,
			builder.NewStock().ProductIDs(queryData.ID).Versions(newestRecord.Version).QuantityGT(queryData.Quantity),
			map[string]any{
				"quantity": gorm.Expr("quantity - ?", queryData.Quantity),
				"version":  newestRecord.Version + 1,
			}); err != nil {
			return err
		}
	}

	//existing := m.unmarshalFromDatabase(dest)
	//
	//// Call the updateFn to determine the new quantities
	//updated, err := updateFn(ctx, existing, data)
	//if err != nil {
	//	return err
	//}
	//
	//// Process each item that needs updating
	//for _, upd := range updated {
	//	// Find the original item to get its version
	//	var record persistent.StockModel
	//	if err := builder.NewStock().ProductIDs(upd.ID).
	//		Fill(tx.Model(&persistent.StockModel{})).
	//		First(&record).Error; err != nil {
	//		return errors.Wrapf(err, "unable to find record for %s", upd.ID)
	//	}
	//
	//	// Update with optimistic locking using version
	//	if err := builder.NewStock().ProductIDs(upd.ID).Versions(record.Version).QuantityGT(upd.Quantity).
	//		Fill(tx.Model(&persistent.StockModel{})).
	//		Updates(map[string]any{
	//			"quantity": gorm.Expr("quantity - ?", upd.Quantity),
	//			"version":  record.Version + 1,
	//		}).Error; err != nil {
	//		return errors.Wrapf(err, "unable to update %s", upd.ID)
	//	}
	//}
	return nil
}
