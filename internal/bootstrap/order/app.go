package order

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
	event_delivery "github.com/mafi020/ecom-golang-micro/internal/delivery/events"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/gRPC/handler"
	grpc_utils "github.com/mafi020/ecom-golang-micro/internal/delivery/gRPC/utils"
	"github.com/mafi020/ecom-golang-micro/internal/infrastructure"
	"github.com/mafi020/ecom-golang-micro/internal/utils"
	orderpb "github.com/mafi020/ecom-golang-micro/proto/order"
	"github.com/mafi020/ecom-golang-micro/rpc_client"
	"google.golang.org/grpc"
)

type OrderApp struct {
	db       *sql.DB
	config   *config.Config
	usecases *Usecases
	rpcPool  *rpc_client.Clients
	broker   *utils.EventBroker
}

func InitializeOrderApp() *OrderApp {
	cfg := config.LoadConfig()
	order_dsn := cfg.PostgresDSN(
		cfg.Postgres.PgOrderUser,
		cfg.Postgres.PgOrderPassword,
		cfg.Postgres.PgOrderHost,
		cfg.Postgres.PgOrderDBName,
		cfg.Postgres.PgOrderPort,
	)

	db := infrastructure.NewPostgresDB(order_dsn, cfg.Postgres.PgOrderDBName)

	if err := infrastructure.RunMigrations(order_dsn, "migrations/order"); err != nil {
		slog.Error("failed to run oder migrations", slog.Any("error", err))
		panic(err)
	}

	rpcPool, err := rpc_client.InitializeClients(rpc_client.Config{
		CatalogAddr: "localhost:" + cfg.Server.CatalogServiceGRPCPort,
		CartAddr:    "localhost:" + cfg.Server.CartServiceGRPCPort,
	})

	if err != nil {
		slog.Error("Microservice cluster initialization blocked: cannot reach Services", slog.Any("error", err))
		panic(err)
	}

	transactor := infrastructure.NewPostgresTransactor(db)

	broker, err := utils.NewEventBroker(cfg.Server.RabbitMqURL)
	if err != nil {
		slog.Error("Failed to initialize RabbitMQ Event Broker", slog.Any("error", err))
		panic(err)
	}

	repos := RegisterRepositories(db)
	usecases := RegisterUsecases(repos, transactor, cfg, rpcPool, broker)

	// 2. Initialize and spawn the background event processing consumer loop
	consumer := event_delivery.NewOrderEventConsumer(broker, usecases.OrderUC)
	if err := consumer.StartListening(); err != nil {
		slog.Error("Failed to initiate background worker processing loops", slog.Any("error", err))
		panic(err)
	}

	return &OrderApp{db: db, config: cfg, usecases: usecases, rpcPool: rpcPool, broker: broker}
}

func (a *OrderApp) RunHTTP() (*http.Server, error) {
	r := gin.Default()

	RegisterHTTPHandlers(r, a.usecases, a.config)

	srv := &http.Server{
		Addr:    ":" + a.config.Server.OrderServiceHTTPPort,
		Handler: r,
	}

	go func() {
		slog.Info("Order HTTP Server running cleanly on ", slog.String("port", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Order HTTP Server failure", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

func (a *OrderApp) ShutdownHTTP(srv *http.Server) error {
	slog.Info("Shutting down Order HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Forced HTTP shutdown failed to release system resources cleanly", slog.Any("error", err))
		return fmt.Errorf("forced Order HTTP shutdown failed: %w", err)
	}

	slog.Info("Order HTTP Server exited cleanly")
	return nil
}

func (a *OrderApp) RunGRPC() (*grpc.Server, error) {

	port := a.config.Server.OrderServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("Failed to lock and bind system TCP socket for gRPC engine", slog.String("port", port), slog.Any("error", err))
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer(
		// Logger for gRPC
		grpc.UnaryInterceptor(grpc_utils.UnaryServerLoggerInterceptor()),
	)

	// Create handler structure passing down needed usecases context
	grpcHandler := handler.NewOrderGRPCHandler(a.usecases.OrderUC)
	orderpb.RegisterOrderServiceServer(srv, grpcHandler)

	go func() {
		slog.Info("Order grpc Server running cleanly on", slog.String("port", port))
		if err := srv.Serve(lis); err != nil {
			slog.Error("Fatal runtime exception thrown during gRPC mesh processing operations", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

// ShutdownGRPC cleans up and terminates the grpc processes gracefully
func (a *OrderApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()
	defer a.rpcPool.Close()
	defer a.broker.Close()
	slog.Info("Shutting down Order grpc server...")

	// GracefulStop waits for all active connections and RPC requests to finish processing
	srv.GracefulStop()

	slog.Info("Order grpc Server exited cleanly")
	return nil
}
