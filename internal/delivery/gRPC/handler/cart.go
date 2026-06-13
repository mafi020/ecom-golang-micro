package handler

import (
	"context"

	"github.com/mafi020/ecom-golang-micro/internal/delivery/gRPC/utils"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
	cartpb "github.com/mafi020/ecom-golang-micro/proto/cart"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type cartUseCase interface {
	GetCart(ctx context.Context, userID int64) (*entity.Cart, error)
	AddItem(ctx context.Context, userID, productID int64, quantity int32) (*entity.Cart, error)
	UpdateItem(ctx context.Context, userID, productID int64, quantity int32) (*entity.Cart, error)
	RemoveItem(ctx context.Context, userID, productID int64) (*entity.Cart, error)
	ClearCart(ctx context.Context, userID int64) error
}

type CartGRPCHandler struct {
	cartpb.UnimplementedCartServiceServer
	uc cartUseCase
}

func NewCartGRPCHandler(uc cartUseCase) *CartGRPCHandler {
	return &CartGRPCHandler{uc: uc}
}

func (h *CartGRPCHandler) GetCart(ctx context.Context, req *cartpb.GetCartRequest) (*cartpb.GetCartResponse, error) {
	cart, err := h.uc.GetCart(ctx, req.GetUserId())
	if err != nil {
		return nil, utils.HandleGRPCError(err) // 🚀 Clean, consistent handling
	}
	return &cartpb.GetCartResponse{Cart: h.toProtoCart(cart)}, nil
}

func (h *CartGRPCHandler) AddItem(ctx context.Context, req *cartpb.AddItemRequest) (*cartpb.CartResponse, error) {
	cart, err := h.uc.AddItem(ctx, req.GetUserId(), req.GetProductId(), req.GetQuantity())
	if err != nil {
		return nil, utils.HandleGRPCError(err)
	}
	return &cartpb.CartResponse{Cart: h.toProtoCart(cart)}, nil
}

func (h *CartGRPCHandler) UpdateItemQuantity(ctx context.Context, req *cartpb.UpdateItemQuantityRequest) (*cartpb.CartResponse, error) {
	cart, err := h.uc.UpdateItem(ctx, req.GetUserId(), req.GetProductId(), req.GetQuantity())
	if err != nil {
		return nil, utils.HandleGRPCError(err)
	}
	return &cartpb.CartResponse{Cart: h.toProtoCart(cart)}, nil
}

func (h *CartGRPCHandler) RemoveItem(ctx context.Context, req *cartpb.RemoveItemRequest) (*cartpb.CartResponse, error) {
	cart, err := h.uc.RemoveItem(ctx, req.GetUserId(), req.GetProductId())
	if err != nil {
		return nil, utils.HandleGRPCError(err)
	}
	return &cartpb.CartResponse{Cart: h.toProtoCart(cart)}, nil
}

func (h *CartGRPCHandler) ClearCart(ctx context.Context, req *cartpb.ClearCartRequest) (*emptypb.Empty, error) {
	if err := h.uc.ClearCart(ctx, req.GetCartId()); err != nil {
		return nil, utils.HandleGRPCError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *CartGRPCHandler) toProtoCart(c *entity.Cart) *cartpb.Cart {
	if c == nil {
		return nil
	}
	protoItems := make([]*cartpb.CartItem, len(c.Items))
	for i, item := range c.Items {
		protoItems[i] = &cartpb.CartItem{
			Id:         item.ID,
			CartId:     item.CartID,
			ProductId:  item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: item.PriceCents,
		}
	}
	return &cartpb.Cart{
		Id:             c.ID,
		UserId:         c.UserID,
		CreatedAt:      timestamppb.New(c.CreatedAt),
		UpdatedAt:      timestamppb.New(c.UpdatedAt),
		Items:          protoItems,
		SubtotalCents:  c.GetSubtotal(),
		TotalItemCount: c.GetTotalItemCount(),
	}
}
