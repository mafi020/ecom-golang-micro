package usecase

import (
	"context"

	"github.com/mafi020/ecom-golang-micro/internal/entity"
	"github.com/mafi020/ecom-golang-micro/internal/utils"
)

type productInterface interface {
	Create(ctx context.Context, product *entity.Product) error
	GetByID(ctx context.Context, id int64) (*entity.Product, error)
	Update(ctx context.Context, id int64, input *entity.UpdateProductInput) (*entity.Product, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params entity.GetProductsParams) ([]entity.Product, int, error)
	BatchUpdate(ctx context.Context, updates map[int64]*entity.UpdateProductInput) error
	GetByIDs(ctx context.Context, ids []int64) ([]entity.Product, error)
}

type productCategoryInterface interface {
	GetCategoryByID(ctx context.Context, id int64) (*entity.Category, error)
}

type ProductUseCase struct {
	productRepo  productInterface
	categoryRepo productCategoryInterface
	broker       *utils.EventBroker
}

func NewProductUseCase(productRepo productInterface, categoryRepo productCategoryInterface, broker *utils.EventBroker) *ProductUseCase {
	return &ProductUseCase{productRepo: productRepo, categoryRepo: categoryRepo, broker: broker}
}

func (uc *ProductUseCase) Create(ctx context.Context, product *entity.Product) error {
	_, err := uc.categoryRepo.GetCategoryByID(ctx, product.CategoryID)
	if err != nil {
		return err
	}
	return uc.productRepo.Create(ctx, product)
}

func (uc *ProductUseCase) GetByID(ctx context.Context, id int64) (*entity.Product, error) {
	return uc.productRepo.GetByID(ctx, id)
}

func (uc *ProductUseCase) Update(ctx context.Context, id int64, input *entity.UpdateProductInput) (*entity.Product, error) {
	return uc.productRepo.Update(ctx, id, input)
}

func (uc *ProductUseCase) UpdateProductStock(ctx context.Context, productID int64, newStock int32) error {
	input := &entity.UpdateProductInput{
		Stock: &newStock,
	}

	// Call your existing business update method cleanly
	_, err := uc.productRepo.Update(ctx, productID, input)
	return err
}

func (uc *ProductUseCase) Delete(ctx context.Context, id int64) error {
	return uc.productRepo.Delete(ctx, id)
}

func (uc *ProductUseCase) List(ctx context.Context, params entity.GetProductsParams) ([]entity.Product, int, error) {
	return uc.productRepo.List(ctx, params)
}

func (uc *ProductUseCase) BatchUpdate(ctx context.Context, updates map[int64]*entity.UpdateProductInput) error {
	return uc.productRepo.BatchUpdate(ctx, updates)
}

func (uc *ProductUseCase) GetByIDs(ctx context.Context, ids []int64) ([]entity.Product, error) {
	return uc.productRepo.GetByIDs(ctx, ids)
}
