package identity

import (
	"database/sql"

	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
)

type Repository struct {
	UserRepo         *infrastructure.PostgresUserRepository
	RefreshTokenRepo *infrastructure.PostgresRefreshTokenRepository
}

func RegisterRepositories(db *sql.DB) *Repository {
	return &Repository{
		UserRepo:         infrastructure.NewPostgresUserRepository(db),
		RefreshTokenRepo: infrastructure.NewPostgresRefreshTokenRepository(db),
	}
}
