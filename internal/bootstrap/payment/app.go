package payment

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
	appLogger := logger.NewJSONLogger()
	slog.SetDefault(appLogger)
	payment_dsn := cfg.PostgresDSN(cfg.Postgres.PgPaymentUser, cfg.Postgres.PgPaymentPassword, cfg.Postgres.PgPaymentHost, cfg.Postgres.PgPaymentDBName, cfg.Postgres.PgPaymentPort)

	db := infrastructure.NewPostgresDB(payment_dsn, cfg.Postgres.PgPaymentDBName)

	if err := infrastructure.RunMigrations(payment_dsn, "migrations/payment"); err != nil {
		slog.Error("failed to run payment migrations", slog.Any("error", err))
		panic(err)
	}

	rpcPool, err := rpc_client.InitializeClients(rpc_client.Config{
		OrderAddr: "localhost:" + cfg.Server.OrderServiceGRPCPort,
	})

	if err != nil {
		slog.Error("Microservice cluster initialization blocked: cannot reach Services", slog.Any("error", err))
		panic(err)
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
		slog.Info("Payment HTTP Server running cleanly on ", slog.String("port", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Payment HTTP Server failure", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

func (a *PaymentApp) ShutdownHTTP(srv *http.Server) error {
	slog.Info("Shutting down Payment HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Forced HTTP shutdown failed to release system resources cleanly", slog.Any("error", err))
		return fmt.Errorf("forced Payment HTTP shutdown failed: %w", err)
	}

	slog.Info("Payment HTTP Server exited cleanly")
	return nil
}

func (a *PaymentApp) RunGRPC() (*grpc.Server, error) {

	port := a.config.Server.PaymentServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("Failed to lock and bind system TCP socket for gRPC engine", slog.String("port", port), slog.Any("error", err))
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer()

	grpcHandler := handler.NewPaymentGRPCHandler(a.usecases.PaymentUC)
	paymentpb.RegisterPaymentServiceServer(srv, grpcHandler)

	go func() {
		slog.Info("Payment grpc Server running cleanly on", slog.String("port", port))
		if err := srv.Serve(lis); err != nil {
			slog.Error("Fatal runtime exception thrown during gRPC mesh processing operations", slog.Any("error", err))
			panic(err)
		}
	}()

	return srv, nil
}

func (a *PaymentApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()
	defer a.rpcPool.Close()
	slog.Info("Shutting down Payment grpc server...")

	srv.GracefulStop()

	slog.Info("Payment grpc Server exited cleanly")
	return nil
}
