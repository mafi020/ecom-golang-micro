package order

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
	orderpb "github.com/mafi020/ecom-golang-micro/proto/order"
	"github.com/mafi020/ecom-golang-micro/rpc_client"
	"google.golang.org/grpc"
)

type OrderApp struct {
	db       *sql.DB
	config   *config.Config
	usecases *Usecases
	rpcPool  *rpc_client.Clients
}

func InitializeOrderApp() *OrderApp {
	cfg := config.LoadConfig()
	order_dsn := cfg.PostgresDSN(cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Host, cfg.Postgres.OrderDBName, cfg.Postgres.Port)

	db := infrastructure.NewPostgresDB(order_dsn, cfg.Postgres.OrderDBName)

	if err := infrastructure.RunMigrations(order_dsn, "migrations/order"); err != nil {
		log.Fatalf("failed to run order migrations: %v", err)
	}

	rpcPool, err := rpc_client.InitializeClients(rpc_client.Config{
		CatalogAddr: "localhost:" + cfg.Server.CatalogServiceGRPCPort,
		CartAddr:    "localhost:" + cfg.Server.CartServiceGRPCPort,
	})

	if err != nil {
		log.Fatalf("Order microservice initialization blocked: cannot reach Catalog service: %v", err)
	}

	transactor := infrastructure.NewPostgresTransactor(db)

	repos := RegisterRepositories(db)
	usecases := RegisterUsecases(repos, transactor, cfg, rpcPool)

	return &OrderApp{db: db, config: cfg, usecases: usecases, rpcPool: rpcPool}
}

func (a *OrderApp) RunHTTP() (*http.Server, error) {
	r := gin.Default()

	RegisterHTTPHandlers(r, a.usecases, a.config)

	srv := &http.Server{
		Addr:    ":" + a.config.Server.OrderServiceHTTPPort,
		Handler: r,
	}

	go func() {
		log.Printf("Order HTTP Server running cleanly on port %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Order HTTP Server failure: %v", err)
		}
	}()

	return srv, nil
}

func (a *OrderApp) ShutdownHTTP(srv *http.Server) error {
	log.Println("Shutting down Order HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced Order HTTP shutdown failed: %w", err)
	}

	log.Println("Order HTTP Server exited cleanly")
	return nil
}

func (a *OrderApp) RunGRPC() (*grpc.Server, error) {

	port := a.config.Server.OrderServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer()

	// Create handler structure passing down needed usecases context
	grpcHandler := handler.NewOrderGRPCHandler(a.usecases.OrderUC)
	orderpb.RegisterOrderServiceServer(srv, grpcHandler)

	go func() {
		log.Printf("Order grpc Server running cleanly on port :%s", port)
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("Order grpc Server failure: %v", err)
		}
	}()

	return srv, nil
}

// ShutdownGRPC cleans up and terminates the grpc processes gracefully
func (a *OrderApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()
	defer a.rpcPool.Close()
	log.Println("Shutting down Order grpc server...")

	// GracefulStop waits for all active connections and RPC requests to finish processing
	srv.GracefulStop()

	log.Println("Order grpc Server exited cleanly")
	return nil
}
