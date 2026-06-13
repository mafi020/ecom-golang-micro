package payment

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
	paymentpb "github.com/mafi020/ecom-golang-micro/proto/payment"
	"github.com/mafi020/ecom-golang-micro/rpc_client"
	"google.golang.org/grpc"
)

type PaymentApp struct {
	db       *sql.DB
	config   *config.Config
	usecases *Usecases
	rpcPool  *rpc_client.Clients
}

func InitializePaymentApp() *PaymentApp {
	cfg := config.LoadConfig()
	payment_dsn := cfg.PostgresDSN(cfg.Postgres.PgPaymentUser, cfg.Postgres.PgPaymentPassword, cfg.Postgres.PgPaymentHost, cfg.Postgres.PgPaymentDBName, cfg.Postgres.PgPaymentPort)

	db := infrastructure.NewPostgresDB(payment_dsn, cfg.Postgres.PgPaymentDBName)

	if err := infrastructure.RunMigrations(payment_dsn, "migrations/payment"); err != nil {
		log.Fatalf("failed to run payment migrations: %v", err)
	}

	rpcPool, err := rpc_client.InitializeClients(rpc_client.Config{
		OrderAddr: "localhost:" + cfg.Server.OrderServiceGRPCPort,
	})

	if err != nil {
		log.Fatalf("Payment	microservice initialization blocked: cannot reach Order service: %v", err)
	}

	transactor := infrastructure.NewPostgresTransactor(db)

	repos := RegisterRepositories(db)
	usecases := RegisterUsecases(repos, transactor, cfg, rpcPool)

	return &PaymentApp{db: db, config: cfg, usecases: usecases, rpcPool: rpcPool}
}

func (a *PaymentApp) RunHTTP() (*http.Server, error) {
	r := gin.Default()

	RegisterHTTPHandlers(r, a.usecases, a.config)

	srv := &http.Server{
		Addr:    ":" + a.config.Server.PaymentServiceHTTPPort,
		Handler: r,
	}

	go func() {
		log.Printf("Payment HTTP Server running cleanly on port %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Payment HTTP Server failure: %v", err)
		}
	}()

	return srv, nil
}

func (a *PaymentApp) ShutdownHTTP(srv *http.Server) error {
	log.Println("Shutting down Payment HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced Payment HTTP shutdown failed: %w", err)
	}

	log.Println("Payment HTTP Server exited cleanly")
	return nil
}

func (a *PaymentApp) RunGRPC() (*grpc.Server, error) {

	port := a.config.Server.PaymentServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer()

	// Create handler structure passing down needed usecases context
	grpcHandler := handler.NewPaymentGRPCHandler(a.usecases.PaymentUC)
	paymentpb.RegisterPaymentServiceServer(srv, grpcHandler)

	go func() {
		log.Printf("Payment grpc Server running cleanly on port :%s", port)
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("Payment grpc Server failure: %v", err)
		}
	}()

	return srv, nil
}

// ShutdownGRPC cleans up and terminates the grpc processes gracefully
func (a *PaymentApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()
	defer a.rpcPool.Close()
	log.Println("Shutting down Payment grpc server...")

	// GracefulStop waits for all active connections and RPC requests to finish processing
	srv.GracefulStop()

	log.Println("Payment grpc Server exited cleanly")
	return nil
}
