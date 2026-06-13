package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/gRPC/handler"
	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
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
	catalog_dsn := cfg.PostgresDSN(cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Host, cfg.Postgres.CatalogDBName, cfg.Postgres.Port)

	db := infrastructure.NewPostgresDB(catalog_dsn, cfg.Postgres.CatalogDBName)

	if err := infrastructure.RunMigrations(catalog_dsn, "migrations/catalog"); err != nil {
		log.Fatalf("failed to run catalog migrations: %v", err)
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
		log.Printf("Catalog HTTP Server running cleanly on port :%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Catalog HTTP Server failure: %v", err)
		}
	}()

	return srv, nil
}

func (a *CatalogApp) ShutdownHTTP(srv *http.Server) error {

	log.Println("Shutting down Catalog HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced Catalog HTTP shutdown failed: %w", err)
	}

	log.Println("Catalog HTTP Server exited cleanly")
	return nil
}

func (a *CatalogApp) RunGRPC() (*grpc.Server, error) {
	port := a.config.Server.CatalogServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer()

	// Create handler structure passing down needed usecases context
	grpcHandler := handler.NewCatalogGRPCServer(a.usecases.ProductUC)
	catalogpb.RegisterCatalogServiceServer(srv, grpcHandler)

	go func() {
		log.Printf("Catalog grpc Server running cleanly on port :%s", port)
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("Catalog grpc Server failure: %v", err)
		}
	}()

	return srv, nil
}

// ShutdownGRPC cleans up and terminates the grpc processes gracefully
func (a *CatalogApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()
	log.Println("Shutting down Catalog grpc server...")

	// GracefulStop waits for all active connections and RPC requests to finish processing
	srv.GracefulStop()

	log.Println("Catalog grpc Server exited cleanly")
	return nil
}
