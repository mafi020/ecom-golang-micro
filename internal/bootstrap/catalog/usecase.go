package catalog

import (
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/usecase"
)

type Usecases struct {
	CategoryUC *usecase.CategoryUseCase
	ProductUC  *usecase.ProductUseCase
}

func RegisterUsecases(repos *Repositories, cfg *config.Config) *Usecases {
	return &Usecases{
		CategoryUC: usecase.NewCategoryUseCase(repos.CategoryRepo),
		ProductUC:  usecase.NewProductUseCase(repos.ProductRepo, repos.CategoryRepo),
	}
}
