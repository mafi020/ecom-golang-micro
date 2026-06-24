package payment

import (
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
	"github.com/mafi020/ecom-golang-micro/internal/usecase"
	"github.com/mafi020/ecom-golang-micro/internal/utils"
	"github.com/mafi020/ecom-golang-micro/rpc_client"
)

type Usecases struct {
	PaymentUC *usecase.PaymentUseCase
}

func RegisterUsecases(repos *Repositories, tm *infrastructure.Transactor, cfg *config.Config, rpcClients *rpc_client.Clients, broker *utils.EventBroker) *Usecases {
	return &Usecases{
		PaymentUC: usecase.NewPaymentUseCase(repos.PaymentRepo, rpcClients.Order, tm, broker),
	}
}
