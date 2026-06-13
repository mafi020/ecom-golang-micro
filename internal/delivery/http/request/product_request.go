package request

type CreateProduct struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	PriceCents  int64   `json:"price_cents" binding:"required,gt=0"`
	Stock       int32   `json:"stock" binding:"required,gte=0"`
	CategoryID  int64   `json:"category_id" binding:"required,gt=0"`
}

type UpdateProductRequest struct {
	Name        *string `json:"name"        binding:"omitempty,min=1"`
	Description *string `json:"description"`
	PriceCents  *int64  `json:"price_cents"       binding:"omitempty,gt=0"`
	Stock       *int32  `json:"stock"       binding:"omitempty,gte=0"`
	CategoryID  *int64  `json:"category_id" binding:"omitempty,gt=0"`
}
