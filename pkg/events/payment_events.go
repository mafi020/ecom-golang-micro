package events

type PaymentCompletedEvent struct {
	OrderID       int64  `json:"order_id"`
	UserID        int64  `json:"user_id"`
	TransactionID string `json:"transaction_id"`
	AmountCents   int64  `json:"amount_cents"`
}
