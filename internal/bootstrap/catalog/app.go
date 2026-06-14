package catalog

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
	catalogpb "github.com/mafi020/ecom-golang-micro/proto/catalog"
	"google.golang.org/grpc"
)

type CatalogApp struct {
	db       *sql.DB
	config   *config.Config
	usecases *Usecases
}

func InitializeCatalogApp() *CatalogApp {
	cfg := config.LoadConfig()
	appLogger := logger.NewJSONLogger()
	slog.SetDefault(appLogger)
	catalog_dsn := cfg.PostgresDSN(cfg.Postgres.PgCatalogUser, cfg.Postgres.PgCatalogPassword, cfg.Postgres.PgCatalogHost, cfg.Postgres.PgCatalogDBName, cfg.Postgres.PgCatalogPort)

	db := infrastructure.NewPostgresDB(catalog_dsn, cfg.Postgres.PgCatalogDBName)

	if err := infrastructure.RunMigrations(catalog_dsn, "migrations/catalog"); err != nil {
		slog.Error("failed to run catalog  migrations", slog.Any("error", err))
		panic(err)
	}

	repos := RegisterRepositories(db)
	usecases := RegisterUsecases(repos, cfg)

	return &CatalogApp{db: db, config: cfg, usecases: usecases}
}

func (a *CatalogApp) RunHTTP() (*http.Server, error) {
	r := gin.Default()

	RegisterHTTPHandlers(r, a.usecases, a.config)

	srv := &http.Server{
		Addr:    ":" + a.config.Server.CatalogServiceHTTPPort,
		Handler: r,
	}

	go func() {
		slog.Info("Catalog HTTP Server running cleanly on ", slog.String("port", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Catalog HTTP Server failure", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

func (a *CatalogApp) ShutdownHTTP(srv *http.Server) error {

	slog.Info("Shutting down Catalog HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Forced HTTP shutdown failed to release system resources cleanly", slog.Any("error", err))
		return fmt.Errorf("forced Catalog HTTP shutdown failed: %w", err)
	}

	slog.Info("Catalog HTTP Server exited cleanly")
	return nil
}

func (a *CatalogApp) RunGRPC() (*grpc.Server, error) {
	port := a.config.Server.CatalogServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("Failed to lock and bind system TCP socket for gRPC engine", slog.String("port", port), slog.Any("error", err))
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer()

	// Create handler structure passing down needed usecases context
	grpcHandler := handler.NewCatalogGRPCServer(a.usecases.ProductUC)
	catalogpb.RegisterCatalogServiceServer(srv, grpcHandler)

	go func() {
		slog.Info("Catalog grpc Server running cleanly on", slog.String("port", port))
		if err := srv.Serve(lis); err != nil {
			slog.Error("Fatal runtime exception thrown during gRPC mesh processing operations", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

// ShutdownGRPC cleans up and terminates the grpc processes gracefully
func (a *CatalogApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()
	slog.Info("Shutting down Catalog grpc server...")

	// GracefulStop waits for all active connections and RPC requests to finish processing
	srv.GracefulStop()

	slog.Info("Catalog grpc Server exited cleanly")
	return nil
}
