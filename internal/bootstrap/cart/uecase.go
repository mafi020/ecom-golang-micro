package cart

import (
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/usecase"
	"github.com/mafi020/ecom-golang-micro/internal/utils"
	catalogpb "github.com/mafi020/ecom-golang-micro/proto/catalog"
)

type Usecases struct {
	CartUC *usecase.CartUseCase
}

func RegisterUsecases(repos *Repositories, cfg *config.Config, catalogClient catalogpb.CatalogServiceClient, broker *utils.EventBroker) *Usecases {
	return &Usecases{
		CartUC: usecase.NewCartUsecase(repos.CartRepo, repos.CartItemRepo, catalogClient, broker),
	}
}
