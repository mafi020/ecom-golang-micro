package cart

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
	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
	"github.com/mafi020/ecom-golang-micro/internal/logger"
	cartpb "github.com/mafi020/ecom-golang-micro/proto/cart"
	"github.com/mafi020/ecom-golang-micro/rpc_client"
	"google.golang.org/grpc"
)

type CartApp struct {
	db       *sql.DB
	config   *config.Config
	usecases *Usecases
	rpcPool  *rpc_client.Clients
}

func InitializeCartApp() *CartApp {
	cfg := config.LoadConfig()
	appLogger := logger.NewJSONLogger()
	slog.SetDefault(appLogger)

	cart_dsn := cfg.PostgresDSN(
		cfg.Postgres.PgCartUser,
		cfg.Postgres.PgCartPassword,
		cfg.Postgres.PgCartHost,
		cfg.Postgres.PgCartDBName,
		cfg.Postgres.PgCartPort,
	)

	db := infrastructure.NewPostgresDB(cart_dsn, cfg.Postgres.PgCartDBName)

	if err := infrastructure.RunMigrations(cart_dsn, "migrations/cart"); err != nil {
		slog.Error("failed to run cart migrations", slog.Any("error", err))
		panic(err)
	}

	rpcPool, err := rpc_client.InitializeClients(rpc_client.Config{
		CatalogAddr: "localhost:" + cfg.Server.CatalogServiceGRPCPort,
	})

	if err != nil {
		slog.Error("Microservice cluster initialization blocked: cannot reach Services", slog.Any("error", err))
		panic(err)
	}

	repos := RegisterRepositories(db)
	usecases := RegisterUsecases(repos, cfg, rpcPool.Catalog)

	return &CartApp{db: db, config: cfg, usecases: usecases, rpcPool: rpcPool}
}

func (a *CartApp) RunHTTP() (*http.Server, error) {
	r := gin.Default()

	RegisterHTTPHandlers(r, a.usecases, a.config)

	srv := &http.Server{
		Addr:    ":" + a.config.Server.CartServiceHTTPPort,
		Handler: r,
	}

	go func() {
		slog.Info("Cart HTTP Server running cleanly on ", slog.String("port", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Cart HTTP Server failure", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

func (a *CartApp) ShutdownHTTP(srv *http.Server) error {

	slog.Info("Shutting down Cart HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Forced HTTP shutdown failed to release system resources cleanly", slog.Any("error", err))
		return fmt.Errorf("forced Cart HTTP shutdown failed: %w", err)
	}

	slog.Info("Cart HTTP Server exited cleanly")
	return nil
}

func (a *CartApp) RunGRPC() (*grpc.Server, error) {
	port := a.config.Server.CartServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("Failed to lock and bind system TCP socket for gRPC engine", slog.String("port", port), slog.Any("error", err))
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer()

	grpcHandler := handler.NewCartGRPCHandler(a.usecases.CartUC)
	cartpb.RegisterCartServiceServer(srv, grpcHandler)

	go func() {
		slog.Info("Cart grpc Server running cleanly on", slog.String("port", port))
		if err := srv.Serve(lis); err != nil {
			slog.Error("Fatal runtime exception thrown during gRPC mesh processing operations", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

// ShutdownGRPC cleans up and terminates the grpc processes gracefully
func (a *CartApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()
	defer a.rpcPool.Close()

	slog.Info("Shutting down Cart grpc server...")

	// GracefulStop waits for all active connections and RPC requests to finish processing
	srv.GracefulStop()

	slog.Info("Cart grpc Server exited cleanly")
	return nil
}
