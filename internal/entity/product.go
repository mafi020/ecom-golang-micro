package entity

import (
	"time"

	"github.com/mafi020/ecom-golang-micro/internal/utils"
)

type Product struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	PriceCents  int64     `json:"price_cents"`
	Stock       int32     `json:"stock"`
	CategoryID  int64     `json:"category_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	//  setup relationship with category
	Category *Category `json:"category,omitempty"`
}

func NewProduct(name string, description *string, price_cents int64, stock int32, categoryID int64) *Product {
	return &Product{
		Name:        name,
		Description: description,
		PriceCents:  price_cents,
		Stock:       stock,
		CategoryID:  categoryID,
	}
}

type UpdateProductInput struct {
	Name        *string
	Description *string
	PriceCents  *int64
	Stock       *int32
	CategoryID  *int64
}

type GetProductsParams struct {
	utils.QueryParams
	CategoryID int64 `form:"category_id"`
}
