package identity

import (
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
	"github.com/mafi020/ecom-golang-micro/internal/usecase"
)

type Usecases struct {
	AuthUC         *usecase.AuthUseCase
	UserUC         *usecase.UserUseCase
	RefreshTokenUC *usecase.AuthUseCase
}

func RegisterUsecases(repos *Repository, tm *infrastructure.Transactor, cfg *config.Config) *Usecases {
	return &Usecases{
		AuthUC:         usecase.NewAuthUsecase(repos.UserRepo, repos.RefreshTokenRepo, cfg),
		UserUC:         usecase.NewUserUsecase(repos.UserRepo),
		RefreshTokenUC: usecase.NewAuthUsecase(repos.UserRepo, repos.RefreshTokenRepo, cfg),
	}
}
