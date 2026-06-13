package order

import (
	"database/sql"

	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
)

type Repositories struct {
	OrderRepo     *infrastructure.PostgresOrderRepository
	OrderItemRepo *infrastructure.PostgresOrderItemRepository
	ProductRepo   *infrastructure.PostgresProductRepository
	CartRepo      *infrastructure.PostgresCartRepository
	CartItemRepo  *infrastructure.PostgresCartItemRepository
}

func RegisterRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		OrderRepo:     infrastructure.NewPostgresOrderRepository(db),
		OrderItemRepo: infrastructure.NewPostgresOrderItemRepository(db),
		ProductRepo:   infrastructure.NewPostgresProductRepository(db),
		CartRepo:      infrastructure.NewPostgresCartRepository(db),
		CartItemRepo:  infrastructure.NewPostgresCartItemRepository(db),
	}
}
