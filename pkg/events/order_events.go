package events

type OrderPlacedEvent struct {
	OrderID      int64           `json:"order_id"`
	UserID       int64           `json:"user_id"`
	TotalPrice   int64           `json:"total_price"`
	StockUpdates map[int64]int32 `json:"stock_updates"`
}
