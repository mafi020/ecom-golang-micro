package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/mafi020/ecom-golang-micro/internal/apperrors"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
	"github.com/mafi020/ecom-golang-micro/internal/logger"
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
}

func NewOrderUseCase(
	orderRepo orderRepo,
	orderItemRepo orderItemRepo,
	catalogClient catalogpb.CatalogServiceClient,
	cartClient cartpb.CartServiceClient,
	tm transactionManager,
) *OrderUseCase {
	return &OrderUseCase{
		orderRepo:     orderRepo,
		orderItemRepo: orderItemRepo,
		catalogClient: catalogClient,
		cartClient:    cartClient,
		tm:            tm,
	}
}

func (uc *OrderUseCase) Checkout(ctx context.Context, userID int64) (*entity.Order, error) {
	log := logger.FromContext(ctx)

	// Step 1: Fetch cart via gRPC — no direct DB access to cart service
	cartCtx, cartCancel := context.WithTimeout(ctx, 2*time.Second)
	defer cartCancel()

	cartResp, err := uc.cartClient.GetCart(cartCtx, &cartpb.GetCartRequest{UserId: userID})
	if err != nil {
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

	// Step 3: Decrement stock in catalog — compensate order on failure
	rpcCtx, rpcCancel := context.WithTimeout(ctx, 2*time.Second)
	defer rpcCancel()

	_, rpcErr := uc.catalogClient.BatchUpdateProducts(rpcCtx, &catalogpb.BatchUpdateProductsRequest{
		Updates: result.gRPCStockPayload,
	})
	if rpcErr != nil {
		compensateCtx, compensateCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer compensateCancel()

		if cancelErr := uc.UpdateStatus(compensateCtx, order.ID, entity.OrderStatusCancelled); cancelErr != nil {
			log.Error("SAGA CRITICAL FAILURE: order created but compensation also failed",
				"order_id", order.ID,
				"rpc_err", rpcErr,
				"cancel_err", cancelErr,
			)
		}
		return nil, fmt.Errorf("checkout failed, order has been cancelled: %w", rpcErr)
	}

	// Step 4: Confirm the order — catalog stock is already decremented
	if err := uc.orderRepo.UpdateStatus(ctx, order.ID, entity.OrderStatusConfirmed); err != nil {
		log.Error("order confirmed in catalog but status update failed in DB",
			"order_id", order.ID,
			"err", err,
		)
	} else {
		order.Status = entity.OrderStatusConfirmed
	}

	// Step 5: Clear the cart — best effort, order is already confirmed
	clearCtx, clearCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer clearCancel()

	_, clearErr := uc.cartClient.ClearCart(clearCtx, &cartpb.ClearCartRequest{CartId: protoCart.GetId()})
	if clearErr != nil {
		log.Error("order confirmed but cart clear failed — cart may be stale",
			"order_id", order.ID,
			"cart_id", protoCart.GetId(),
			"err", clearErr,
		)
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
