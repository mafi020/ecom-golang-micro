package order

import (
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
	"github.com/mafi020/ecom-golang-micro/internal/usecase"
	"github.com/mafi020/ecom-golang-micro/rpc_client"
)

type Usecases struct {
	OrderUC *usecase.OrderUseCase
}

func RegisterUsecases(repos *Repositories, tm *infrastructure.Transactor, cfg *config.Config, rpcClients *rpc_client.Clients) *Usecases {
	return &Usecases{
		OrderUC: usecase.NewOrderUseCase(repos.OrderRepo, repos.OrderItemRepo, rpcClients.Catalog, rpcClients.Cart, tm),
	}
}
