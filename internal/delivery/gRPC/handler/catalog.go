package handler

import (
	"context"

	"github.com/mafi020/ecom-golang-micro/internal/entity"
	catalogpb "github.com/mafi020/ecom-golang-micro/proto/catalog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductUsecase interface {
	GetByID(ctx context.Context, id int64) (*entity.Product, error)
	GetByIDs(ctx context.Context, ids []int64) ([]entity.Product, error)
	BatchUpdate(ctx context.Context, updates map[int64]*entity.UpdateProductInput) error
}

type CatalogGRPCServer struct {
	catalogpb.UnimplementedCatalogServiceServer
	productUC ProductUsecase
}

func NewCatalogGRPCServer(uc ProductUsecase) *CatalogGRPCServer {
	return &CatalogGRPCServer{productUC: uc}
}

func (s *CatalogGRPCServer) GetProduct(ctx context.Context, req *catalogpb.GetProductRequest) (*catalogpb.GetProductResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid product id")
	}

	storedProduct, err := s.productUC.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve product: %v", err)
	}

	return &catalogpb.GetProductResponse{
		Product: &catalogpb.Product{
			Id:         storedProduct.ID,
			Name:       storedProduct.Name,
			PriceCents: storedProduct.PriceCents,
			Stock:      int32(storedProduct.Stock),
		},
	}, nil
}

func (s *CatalogGRPCServer) BatchGetProducts(ctx context.Context, req *catalogpb.BatchGetProductsRequest) (*catalogpb.BatchGetProductsResponse, error) {
	if len(req.GetIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "product ids cannot be empty")
	}

	products, err := s.productUC.GetByIDs(ctx, req.GetIds())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve products: %v", err)
	}

	if len(products) == 0 {
		return nil, status.Error(codes.NotFound, "none of the requested products were found")
	}

	protoProducts := make([]*catalogpb.Product, 0, len(products))
	for _, p := range products {
		protoProducts = append(protoProducts, &catalogpb.Product{
			Id:         p.ID,
			Name:       p.Name,
			PriceCents: p.PriceCents,
			Stock:      int32(p.Stock),
		})
	}

	return &catalogpb.BatchGetProductsResponse{Products: protoProducts}, nil
}

func (s *CatalogGRPCServer) BatchUpdateProducts(ctx context.Context, req *catalogpb.BatchUpdateProductsRequest) (*emptypb.Empty, error) {
	updates := req.GetUpdates()
	if len(updates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "updates mapping matrix cannot be empty")
	}

	updateMap := make(map[int64]*entity.UpdateProductInput, len(updates))
	for id, payload := range updates {
		if payload == nil {
			return nil, status.Errorf(codes.InvalidArgument, "payload for product ID %d cannot be nil", id)
		}

		if id <= 0 {
			return nil, status.Error(codes.InvalidArgument, "invalid product id in updates")
		}

		input := &entity.UpdateProductInput{
			Name:        payload.Name,
			Description: payload.Description,
			PriceCents:  payload.PriceCents,
		}

		if payload.Stock != nil {
			stock := (*payload.Stock)
			input.Stock = &stock
		}

		updateMap[id] = input
	}

	if err := s.productUC.BatchUpdate(ctx, updateMap); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to batch update products: %v", err)
	}

	return &emptypb.Empty{}, nil
}
