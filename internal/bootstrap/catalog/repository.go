package catalog

import (
	"database/sql"

	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
)

type Repositories struct {
	CategoryRepo *infrastructure.PostgresCategoryRepository
	ProductRepo  *infrastructure.PostgresProductRepository
}

func RegisterRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		CategoryRepo: infrastructure.NewPostgresCategoryRepository(db),
		ProductRepo:  infrastructure.NewPostgresProductRepository(db),
	}
}
