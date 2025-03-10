package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQL struct {
	db *gorm.DB
}

type StockModel struct {
	ID        int64     `gorm:"column:id"`
	ProductID string    `gorm:"column:product_id"`
	Quantity  int32     `gorm:"column:quantity"`
	Version   int64     `gorm:"column:version"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdateAt  time.Time `gorm:"column:updated_at"`
}

func NewMySQL() *MySQL {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		viper.GetString("mysql.user"),
		viper.GetString("mysql.password"),
		viper.GetString("mysql.host"),
		viper.GetString("mysql.port"),
		viper.GetString("mysql.dbname"),
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Panicf("connect to mysql failed, err=%v", err)
	}
	return &MySQL{db: db}
}

func NewMySQLWithDB(db *gorm.DB) *MySQL {
	return &MySQL{db: db}
}

func (StockModel) TableName() string {
	return "o_stock"
}

func (m *StockModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.UpdateAt = time.Now()
	return nil
}

func (d MySQL) StartTransaction(f func(tx *gorm.DB) error) error {
	return d.db.Transaction(f)
}

func (d MySQL) BatchGetStockByID(ctx context.Context, productIDs []string) ([]StockModel, error) {
	var result []StockModel
	err := d.db.WithContext(ctx).Where("product_id IN ?", productIDs).Find(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d MySQL) Create(ctx context.Context, create *StockModel) error {
	return d.db.WithContext(ctx).Create(create).Error
}
