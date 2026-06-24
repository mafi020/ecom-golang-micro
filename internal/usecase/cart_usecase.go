package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/mafi020/ecom-golang-micro/internal/apperrors"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
	"github.com/mafi020/ecom-golang-micro/internal/utils"
	catalogpb "github.com/mafi020/ecom-golang-micro/proto/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type cartRepo interface {
	GetOrCreateCart(ctx context.Context, userID int64) (*entity.Cart, error)
	GetCartByUserID(ctx context.Context, userID int64) (*entity.Cart, error)
}

type cartItemRepo interface {
	AddItem(ctx context.Context, cartID int64, item *entity.CartItem) (*entity.CartItem, error)
	UpdateItemQuantity(ctx context.Context, cartID, productID int64, quantity int32) (*entity.CartItem, error)
	RemoveItem(ctx context.Context, cartID, productID int64) error
	ClearCart(ctx context.Context, cartID int64) error
}

type cartProductRepo interface {
	GetByID(ctx context.Context, id int64) (*entity.Product, error)
}

type CartUseCase struct {
	cartRepo        cartRepo
	cartItemRepo    cartItemRepo
	cartProductRepo cartProductRepo
	catalogClient   catalogpb.CatalogServiceClient
	broker          *utils.EventBroker
}

func NewCartUsecase(
	cartRepo cartRepo,
	cartItemRepo cartItemRepo,
	catalogClient catalogpb.CatalogServiceClient,
	broker *utils.EventBroker,
) *CartUseCase {
	return &CartUseCase{
		cartRepo:      cartRepo,
		cartItemRepo:  cartItemRepo,
		catalogClient: catalogClient,
		broker:        broker,
	}
}

func (uc *CartUseCase) GetCart(ctx context.Context, userID int64) (*entity.Cart, error) {
	return uc.cartRepo.GetCartByUserID(ctx, userID)
}

func (uc *CartUseCase) AddItem(ctx context.Context, userID, productID int64, quantity int32) (*entity.Cart, error) {
	cart, product, totalQuantityRequested, err := uc.prepareAndValidateStock(ctx, userID, productID, quantity, false)
	if err != nil {
		return nil, err
	}

	_, err = uc.cartItemRepo.AddItem(ctx, cart.ID, &entity.CartItem{
		ProductID:  productID,
		Quantity:   totalQuantityRequested,
		PriceCents: product.PriceCents,
	})

	if err != nil {
		return nil, err
	}

	return uc.cartRepo.GetCartByUserID(ctx, userID)
}

func (uc *CartUseCase) UpdateItem(ctx context.Context, userID, productID int64, quantity int32) (*entity.Cart, error) {
	cart, _, totalQuantityRequested, err := uc.prepareAndValidateStock(ctx, userID, productID, quantity, true)
	if err != nil {
		return nil, err
	}

	_, err = uc.cartItemRepo.UpdateItemQuantity(ctx, cart.ID, productID, totalQuantityRequested)
	if err != nil {
		return nil, err
	}

	return uc.cartRepo.GetCartByUserID(ctx, userID)
}

func (uc *CartUseCase) RemoveItem(ctx context.Context, userID, productID int64) (*entity.Cart, error) {
	cart, err := uc.cartRepo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := uc.cartItemRepo.RemoveItem(ctx, cart.ID, productID); err != nil {
		return nil, err
	}

	return uc.cartRepo.GetCartByUserID(ctx, userID)
}

func (uc *CartUseCase) ClearCart(ctx context.Context, userID int64) error {
	cart, err := uc.cartRepo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return err
	}
	return uc.cartItemRepo.ClearCart(ctx, cart.ID)
}

func (uc *CartUseCase) prepareAndValidateStock(
	ctx context.Context,
	userID, productID int64,
	inputQuantity int32,
	isAbsoluteUpdate bool,
) (*entity.Cart, *entity.Product, int32, error) {

	rpcCtx, rpcCancel := context.WithTimeout(ctx, 2*time.Second)
	defer rpcCancel()

	resp, rpcErr := uc.catalogClient.GetProduct(rpcCtx, &catalogpb.GetProductRequest{Id: productID})
	if rpcErr != nil {
		if st, ok := status.FromError(rpcErr); ok && st.Code() == codes.NotFound {
			return nil, nil, 0, &apperrors.NotFoundError{Resource: "product"}
		}
		return nil, nil, 0, fmt.Errorf("catalog microservice unavailable: %w", rpcErr)
	}

	protoProduct := resp.GetProduct()

	// 3. Instantiate an internal ephemeral domain entity map using explicit types
	// (Note: PriceCents matches perfectly because we switched proto definition to int64)
	product := &entity.Product{
		ID:         protoProduct.GetId(),
		Name:       protoProduct.GetName(),
		PriceCents: protoProduct.GetPriceCents(),
		Stock:      protoProduct.GetStock(),
	}

	// 4. Fetch or generate customer shopping cart values
	cart, err := uc.cartRepo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, nil, 0, err
	}

	existingQuantityInCart := int32(0)
	for _, item := range cart.Items {
		if item.ProductID == productID {
			existingQuantityInCart = item.Quantity
			break
		}
	}

	// 5. Compute structural checkout demands
	totalQuantityRequested := inputQuantity
	if !isAbsoluteUpdate {
		totalQuantityRequested = existingQuantityInCart + inputQuantity
	}

	// 6. Cross-reference stock data using the fresh, typesafe network integers
	if product.Stock < totalQuantityRequested {
		return nil, nil, 0, &apperrors.BadRequestError{
			Errors: map[string]string{"quantity": "insufficient stock"},
		}
	}

	return cart, product, totalQuantityRequested, nil
}
