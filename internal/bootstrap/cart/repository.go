package cart

import (
	"database/sql"

	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
)

type Repositories struct {
	CartRepo     *infrastructure.PostgresCartRepository
	CartItemRepo *infrastructure.PostgresCartItemRepository
	ProductRepo  *infrastructure.PostgresProductRepository
}

func RegisterRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		CartRepo:     infrastructure.NewPostgresCartRepository(db),
		CartItemRepo: infrastructure.NewPostgresCartItemRepository(db),
		ProductRepo:  infrastructure.NewPostgresProductRepository(db),
	}
}
