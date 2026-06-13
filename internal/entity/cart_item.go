package entity

type CartItem struct {
	ID         int64 `json:"id"`
	CartID     int64 `json:"cart_id"`
	ProductID  int64 `json:"product_id"`
	Quantity   int32 `json:"quantity"`
	PriceCents int64 `json:"price_cents"`
}
