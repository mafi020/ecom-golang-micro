package cart

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
	cart_dsn := cfg.PostgresDSN(cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Host, cfg.Postgres.CartDBName, cfg.Postgres.Port)

	db := infrastructure.NewPostgresDB(cart_dsn, cfg.Postgres.CartDBName)

	if err := infrastructure.RunMigrations(cart_dsn, "migrations/cart"); err != nil {
		log.Fatalf("failed to run cart migrations: %v", err)
	}

	rpcPool, err := rpc_client.InitializeClients(rpc_client.Config{
		CatalogAddr: "localhost:" + cfg.Server.CatalogServiceGRPCPort,
	})

	if err != nil {
		log.Fatalf("Cart microservice initialization blocked: cannot reach Catalog service: %v", err)
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
		log.Printf("Cart HTTP Server running cleanly on port %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Cart HTTP Server failure: %v", err)
		}
	}()

	return srv, nil
}

func (a *CartApp) ShutdownHTTP(srv *http.Server) error {

	log.Println("Shutting down Cart HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced Cart HTTP shutdown failed: %w", err)
	}

	log.Println("Cart HTTP Server exited cleanly")
	return nil
}

func (a *CartApp) RunGRPC() (*grpc.Server, error) {
	port := a.config.Server.CartServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer()

	grpcHandler := handler.NewCartGRPCHandler(a.usecases.CartUC)
	cartpb.RegisterCartServiceServer(srv, grpcHandler)

	go func() {
		log.Printf("Cart grpc Server running cleanly on port :%s", port)
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("Cart grpc Server failure: %v", err)
		}
	}()

	return srv, nil
}

// ShutdownGRPC cleans up and terminates the grpc processes gracefully
func (a *CartApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()
	defer a.rpcPool.Close()

	log.Println("Shutting down Cart grpc server...")

	// GracefulStop waits for all active connections and RPC requests to finish processing
	srv.GracefulStop()

	log.Println("Cart grpc Server exited cleanly")
	return nil
}
