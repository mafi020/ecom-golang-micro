package identity

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/gRPC/handler"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/gRPC/utils"
	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
	identitypb "github.com/mafi020/ecom-golang-micro/proto/identity"
	"google.golang.org/grpc"
)

type IdentityApp struct {
	db       *sql.DB
	config   *config.Config
	usecases *Usecases
}

func InitializeIdentityApp() *IdentityApp {
	cfg := config.LoadConfig()

	identity_dsn := cfg.PostgresDSN(cfg.Postgres.PgIdentityUser,
		cfg.Postgres.PgIdentityPassword,
		cfg.Postgres.PgIdentityHost,
		cfg.Postgres.PgIdentityDBName,
		cfg.Postgres.PgIdentityPort,
	)

	db := infrastructure.NewPostgresDB(identity_dsn, cfg.Postgres.PgIdentityDBName)

	if err := infrastructure.RunMigrations(identity_dsn, "migrations/identity"); err != nil {
		slog.Error("failed to run cart migrations", slog.Any("error", err))
		panic(err)
	}

	transactor := infrastructure.NewPostgresTransactor(db)

	repos := RegisterRepositories(db)
	usecases := RegisterUsecases(repos, transactor, cfg)

	return &IdentityApp{db: db, config: cfg, usecases: usecases}
}

func (a *IdentityApp) RunHTTP() (*http.Server, error) {
	r := gin.Default()

	RegisterHTTPHandlers(r, a.usecases, a.config)

	srv := &http.Server{
		Addr:    ":" + a.config.Server.IdentityServiceHTTPPort,
		Handler: r,
	}

	go func() {
		slog.Info("Identity HTTP Server running cleanly on ", slog.String("port", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Identity HTTP Server failure", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

func (a *IdentityApp) ShutdownHTTP(srv *http.Server) error {
	slog.Info("Shutting down Identity HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Forced HTTP shutdown failed to release system resources cleanly", slog.Any("error", err))
		return fmt.Errorf("forced Identity HTTP shutdown failed: %w", err)
	}

	slog.Info("Identity HTTP Server exited cleanly")
	return nil
}

func (a *IdentityApp) RunGRPC() (*grpc.Server, error) {

	port := a.config.Server.IdentityServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("Failed to lock and bind system TCP socket for gRPC engine", slog.String("port", port), slog.Any("error", err))
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer(
		// Logger for gRPC
		grpc.UnaryInterceptor(utils.UnaryServerLoggerInterceptor()),
	)

	// Create handler structure passing down needed usecases context
	grpcHandler := handler.NewIdentityGRPCHandler(a.usecases.UserUC)
	identitypb.RegisterIdentityServiceServer(srv, grpcHandler)

	go func() {
		slog.Info("Identity grpc Server running cleanly on", slog.String("port", port))
		if err := srv.Serve(lis); err != nil {
			slog.Error("Fatal runtime exception thrown during gRPC mesh processing operations", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

// ShutdownGRPC cleans up and terminates the grpc processes gracefully
func (a *IdentityApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()

	slog.Info("Shutting down Identity grpc server...")

	// GracefulStop waits for all active connections and RPC requests to finish processing
	srv.GracefulStop()

	slog.Info("Identity grpc Server exited cleanly")
	return nil
}
