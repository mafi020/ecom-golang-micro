package payment

import (
	"database/sql"

	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
)

type Repositories struct {
	PaymentRepo *infrastructure.PostgresPaymentRepository
	OrderRepo   *infrastructure.PostgresOrderRepository
}

func RegisterRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		PaymentRepo: infrastructure.NewPostgresPaymentRepository(db),
		OrderRepo:   infrastructure.NewPostgresOrderRepository(db),
	}
}
