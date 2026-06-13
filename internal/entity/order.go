package entity

import (
	"slices"
	"time"

	"github.com/mafi020/ecom-golang-micro/internal/utils"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Updated matrix to support Saga orchestrations and forward-only chronological states
var validTransitions = map[OrderStatus][]OrderStatus{
	// Pending can advance to Confirmed (grpc success) or drop directly to Cancelled (grpc failure)
	OrderStatusPending: {OrderStatusConfirmed, OrderStatusCancelled},

	// Once stock is locked, the order can be paid for, or cancelled by the user/admin
	OrderStatusConfirmed: {OrderStatusPaid, OrderStatusCancelled},

	// Once paid, it progresses down the fulfillment line, or enters a cancellation/refund loop
	OrderStatusPaid: {OrderStatusShipped, OrderStatusCancelled},

	// Post-shipping routes run cleanly forward to delivery and completion boundaries
	OrderStatusShipped:   {OrderStatusDelivered, OrderStatusCancelled},
	OrderStatusDelivered: {OrderStatusCompleted},
	OrderStatusCompleted: {},
	OrderStatusCancelled: {},
}

func (s OrderStatus) CanTransitionTo(next OrderStatus) bool {
	allowed, ok := validTransitions[s]
	if !ok {
		return false
	}
	return slices.Contains(allowed, next)
}

type Order struct {
	ID         int64       `json:"id"`
	UserID     int64       `json:"user_id"`
	Status     OrderStatus `json:"status" default:"pending"`
	TotalPrice int64       `json:"total_price"`

	// PROVEN FIX: Logistics details live safely here inside the Order domain
	CourierPartner *string    `json:"courier_partner,omitempty"`
	TrackingNumber *string    `json:"tracking_number,omitempty"`
	ShippedAt      *time.Time `json:"shipped_at,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	OrderItems []OrderItem `json:"order_items,omitempty"`
	User       *User       `json:"user,omitempty"`
	Payment    *Payment    `json:"payment,omitempty"`
}

type GetOrdersParams struct {
	utils.QueryParams
	Status string `form:"status"`
}
