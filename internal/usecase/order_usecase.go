package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mafi020/ecom-golang-micro/internal/apperrors"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
	"github.com/mafi020/ecom-golang-micro/internal/logger"
	"github.com/mafi020/ecom-golang-micro/internal/utils"
	"github.com/mafi020/ecom-golang-micro/pkg/events"
	cartpb "github.com/mafi020/ecom-golang-micro/proto/cart"
	catalogpb "github.com/mafi020/ecom-golang-micro/proto/catalog"
)

type orderRepo interface {
	CreateOrder(ctx context.Context, order *entity.Order) error
	GetOrderByID(ctx context.Context, id, userID int64) (*entity.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int64, params entity.GetOrdersParams) ([]entity.Order, int, error)
	UpdateStatus(ctx context.Context, id int64, status entity.OrderStatus) error
}

type orderItemRepo interface {
	CreateOrderItems(ctx context.Context, orderID int64, items []entity.OrderItem) ([]entity.OrderItem, error)
}

type transactionManager interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}

type OrderUseCase struct {
	orderRepo     orderRepo
	orderItemRepo orderItemRepo
	catalogClient catalogpb.CatalogServiceClient
	cartClient    cartpb.CartServiceClient
	tm            transactionManager
	broker        *utils.EventBroker
}

func NewOrderUseCase(
	orderRepo orderRepo,
	orderItemRepo orderItemRepo,
	catalogClient catalogpb.CatalogServiceClient,
	cartClient cartpb.CartServiceClient,
	tm transactionManager,
	broker *utils.EventBroker,
) *OrderUseCase {
	return &OrderUseCase{
		orderRepo:     orderRepo,
		orderItemRepo: orderItemRepo,
		catalogClient: catalogClient,
		cartClient:    cartClient,
		tm:            tm,
		broker:        broker,
	}
}

func (uc *OrderUseCase) Checkout(ctx context.Context, userID int64) (*entity.Order, error) {
	log := logger.FromContext(ctx)

	// Step 1: Fetch cart via gRPC — no direct DB access to cart service
	cartCtx, cartCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cartCancel()

	cartResp, err := uc.cartClient.GetCart(cartCtx, &cartpb.GetCartRequest{UserId: userID})
	if err != nil {
		slog.Info("failed to fetch cart", slog.Any("err", err))
		return nil, fmt.Errorf("failed to fetch cart: %w", err)
	}

	protoCart := cartResp.GetCart()
	if len(protoCart.GetItems()) == 0 {
		return nil, &apperrors.ValidationError{Errors: map[string]string{"cart": "cart is empty"}}
	}

	// Map proto cart items to internal entity for prepareOrderItems
	cartItems := make([]entity.CartItem, len(protoCart.GetItems()))
	for i, item := range protoCart.GetItems() {
		cartItems[i] = entity.CartItem{
			ProductID:  item.GetProductId(),
			Quantity:   item.GetQuantity(),
			PriceCents: item.GetPriceCents(),
		}
	}

	result, err := uc.prepareOrderItems(ctx, cartItems)
	if err != nil {
		return nil, err
	}

	// Step 2: Local transaction — order rows only, no network calls inside
	var order *entity.Order
	err = uc.tm.WithinTransaction(ctx, func(txCtx context.Context) error {
		order = &entity.Order{
			UserID:     userID,
			Status:     entity.OrderStatusPending,
			TotalPrice: result.totalPrice,
			OrderItems: result.items,
		}
		if err := uc.orderRepo.CreateOrder(txCtx, order); err != nil {
			return err
		}
		updatedItems, err := uc.orderItemRepo.CreateOrderItems(txCtx, order.ID, result.items)
		if err != nil {
			return err
		}
		order.OrderItems = updatedItems
		return nil
	})
	if err != nil {
		return nil, err
	}

	stockPayload := make(map[int64]int32)
	for productID, updateData := range result.gRPCStockPayload {
		if updateData != nil && updateData.Stock != nil {
			stockPayload[productID] = *updateData.Stock
		}
	}

	orderPlacedEvent := events.OrderPlacedEvent{
		OrderID:      order.ID,
		UserID:       userID,
		TotalPrice:   order.TotalPrice,
		StockUpdates: stockPayload,
	}

	// Broadcast event over RabbitMQ wire. Catalog service catches it and writes local DB updates.
	if err := uc.broker.Publish(ctx, "order.placed", orderPlacedEvent); err != nil {
		log.Error("Order ledger written to database, but async queue notification failed", slog.Any("error", err))
	}

	return order, nil

}

type checkoutResult struct {
	totalPrice       int64
	items            []entity.OrderItem
	gRPCStockPayload map[int64]*catalogpb.UpdateProductPayload
}

func (uc *OrderUseCase) prepareOrderItems(ctx context.Context, cartItems []entity.CartItem) (*checkoutResult, error) {
	productIDs := make([]int64, len(cartItems))
	for i, item := range cartItems {
		productIDs[i] = item.ProductID
	}

	rpcCtx, rpcCancel := context.WithTimeout(ctx, 2*time.Second)
	defer rpcCancel()

	resp, err := uc.catalogClient.BatchGetProducts(rpcCtx, &catalogpb.BatchGetProductsRequest{Ids: productIDs})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product batch from catalog: %w", err)
	}

	productMap := make(map[int64]*catalogpb.Product, len(resp.GetProducts()))
	for _, p := range resp.GetProducts() {
		productMap[p.GetId()] = p
	}

	var totalPrice int64
	items := make([]entity.OrderItem, len(cartItems))
	gRPCStockPayload := make(map[int64]*catalogpb.UpdateProductPayload, len(cartItems))

	for i, cartItem := range cartItems {
		product, ok := productMap[cartItem.ProductID]
		if !ok {
			return nil, &apperrors.NotFoundError{Resource: "product"}
		}

		if product.GetStock() < cartItem.Quantity {
			return nil, &apperrors.ValidationError{Errors: map[string]string{
				"stock": fmt.Sprintf("insufficient stock for product %d: have %d, want %d",
					product.GetId(), product.GetStock(), cartItem.Quantity),
			}}
		}

		items[i] = entity.OrderItem{
			ProductID:  cartItem.ProductID,
			Quantity:   cartItem.Quantity,
			PriceCents: product.GetPriceCents(),
		}
		totalPrice += product.GetPriceCents() * int64(cartItem.Quantity)

		newStock := product.GetStock() - cartItem.Quantity
		gRPCStockPayload[product.GetId()] = &catalogpb.UpdateProductPayload{
			Stock: &newStock,
		}
	}

	return &checkoutResult{
		totalPrice:       totalPrice,
		items:            items,
		gRPCStockPayload: gRPCStockPayload,
	}, nil
}

func (uc *OrderUseCase) GetOrderByID(ctx context.Context, id, userID int64) (*entity.Order, error) {
	return uc.orderRepo.GetOrderByID(ctx, id, userID)
}

func (uc *OrderUseCase) GetOrdersByUserID(ctx context.Context, userID int64, params entity.GetOrdersParams) ([]entity.Order, int, error) {
	return uc.orderRepo.GetOrdersByUserID(ctx, userID, params)
}

func (uc *OrderUseCase) UpdateStatus(ctx context.Context, orderID int64, status entity.OrderStatus) error {
	return uc.orderRepo.UpdateStatus(ctx, orderID, status)
}
