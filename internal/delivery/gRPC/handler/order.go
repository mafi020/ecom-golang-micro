package handler

import (
	"context"

	"github.com/mafi020/ecom-golang-micro/internal/delivery/gRPC/utils"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
	orderpb "github.com/mafi020/ecom-golang-micro/proto/order"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type orderUseCase interface {
	Checkout(ctx context.Context, userID int64) (*entity.Order, error)
	GetOrderByID(ctx context.Context, id, userID int64) (*entity.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int64, params entity.GetOrdersParams) ([]entity.Order, int, error)
	UpdateStatus(ctx context.Context, orderID int64, status entity.OrderStatus) error
}

type OrderGRPCHandler struct {
	orderpb.UnimplementedOrderServiceServer
	uc orderUseCase
}

func NewOrderGRPCHandler(uc orderUseCase) *OrderGRPCHandler {
	return &OrderGRPCHandler{uc: uc}
}

func (h *OrderGRPCHandler) Checkout(ctx context.Context, req *orderpb.CheckoutRequest) (*orderpb.CheckoutResponse, error) {
	order, err := h.uc.Checkout(ctx, req.GetUserId())
	if err != nil {
		return nil, utils.HandleGRPCError(err) // 🚀 Clean, consistent handling
	}
	return &orderpb.CheckoutResponse{Order: h.toProtoOrder(order)}, nil
}

func (h *OrderGRPCHandler) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	order, err := h.uc.GetOrderByID(ctx, req.GetId(), req.GetUserId())
	if err != nil {
		return nil, utils.HandleGRPCError(err) // 🚀 Clean, consistent handling
	}
	return &orderpb.GetOrderResponse{Order: h.toProtoOrder(order)}, nil
}

func (h *OrderGRPCHandler) GetOrders(ctx context.Context, req *orderpb.GetOrdersRequest) (*orderpb.GetOrdersResponse, error) {
	params := entity.GetOrdersParams{
		Status: req.GetStatus(),
	}
	params.Page = int(req.GetPage())
	params.Limit = int(req.GetLimit())

	orders, total, err := h.uc.GetOrdersByUserID(ctx, req.GetUserId(), params)
	if err != nil {
		return nil, utils.HandleGRPCError(err) // 🚀 Clean, consistent handling
	}

	protoOrders := make([]*orderpb.Order, len(orders))
	for i, o := range orders {
		protoOrders[i] = h.toProtoOrder(&o)
	}

	return &orderpb.GetOrdersResponse{
		Orders:     protoOrders,
		TotalCount: int32(total),
	}, nil
}

func (h *OrderGRPCHandler) UpdateStatus(ctx context.Context, req *orderpb.UpdateStatusRequest) (*orderpb.UpdateStatusResponse, error) {
	domainStatus := h.toDomainStatus(req.GetStatus())
	err := h.uc.UpdateStatus(ctx, req.GetOrderId(), domainStatus)
	if err != nil {
		return nil, utils.HandleGRPCError(err) // 🚀 Clean, consistent handling
	}
	return &orderpb.UpdateStatusResponse{UpdatedStatus: req.GetStatus()}, nil
}

// ── DATA TRANSFORMERS ────────────────────────────────────────────────────────

func (h *OrderGRPCHandler) toProtoOrder(o *entity.Order) *orderpb.Order {
	if o == nil {
		return nil
	}

	items := make([]*orderpb.OrderItem, len(o.OrderItems))
	for i, item := range o.OrderItems {
		items[i] = &orderpb.OrderItem{
			Id:         item.ID,
			OrderId:    item.OrderID,
			ProductId:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: item.PriceCents,
			CreatedAt:  timestamppb.New(item.CreatedAt),
			UpdatedAt:  timestamppb.New(item.UpdatedAt),
		}
	}

	protoOrder := &orderpb.Order{
		Id:             o.ID,
		UserId:         o.UserID,
		Status:         h.toProtoStatus(o.Status),
		TotalPrice:     o.TotalPrice,
		CourierPartner: o.CourierPartner,
		TrackingNumber: o.TrackingNumber,
		CreatedAt:      timestamppb.New(o.CreatedAt),
		UpdatedAt:      timestamppb.New(o.UpdatedAt),
		OrderItems:     items,
	}

	if o.ShippedAt != nil {
		protoOrder.ShippedAt = timestamppb.New(*o.ShippedAt)
	}
	if o.DeliveredAt != nil {
		protoOrder.DeliveredAt = timestamppb.New(*o.DeliveredAt)
	}

	return protoOrder
}

func (h *OrderGRPCHandler) toProtoStatus(s entity.OrderStatus) orderpb.OrderStatus {
	switch s {
	case entity.OrderStatusPending:
		return orderpb.OrderStatus_ORDER_STATUS_PENDING
	case entity.OrderStatusConfirmed:
		return orderpb.OrderStatus_ORDER_STATUS_CONFIRMED
	case entity.OrderStatusPaid:
		return orderpb.OrderStatus_ORDER_STATUS_PAID
	case entity.OrderStatusShipped:
		return orderpb.OrderStatus_ORDER_STATUS_SHIPPED
	case entity.OrderStatusDelivered:
		return orderpb.OrderStatus_ORDER_STATUS_DELIVERED
	case entity.OrderStatusCompleted:
		return orderpb.OrderStatus_ORDER_STATUS_COMPLETED
	case entity.OrderStatusCancelled:
		return orderpb.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return orderpb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func (h *OrderGRPCHandler) toDomainStatus(s orderpb.OrderStatus) entity.OrderStatus {
	switch s {
	case orderpb.OrderStatus_ORDER_STATUS_PENDING:
		return entity.OrderStatusPending
	case orderpb.OrderStatus_ORDER_STATUS_CONFIRMED:
		return entity.OrderStatusConfirmed
	case orderpb.OrderStatus_ORDER_STATUS_PAID:
		return entity.OrderStatusPaid
	case orderpb.OrderStatus_ORDER_STATUS_SHIPPED:
		return entity.OrderStatusShipped
	case orderpb.OrderStatus_ORDER_STATUS_DELIVERED:
		return entity.OrderStatusDelivered
	case orderpb.OrderStatus_ORDER_STATUS_COMPLETED:
		return entity.OrderStatusCompleted
	case orderpb.OrderStatus_ORDER_STATUS_CANCELLED:
		return entity.OrderStatusCancelled
	default:
		return entity.OrderStatusPending
	}
}
