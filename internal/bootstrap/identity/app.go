package identity

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
	identity_dsn := cfg.PostgresDSN(cfg.Postgres.PgIdentityUser, cfg.Postgres.PgIdentityPassword, cfg.Postgres.PgIdentityHost, cfg.Postgres.PgIdentityDBName, cfg.Postgres.PgIdentityPort)

	db := infrastructure.NewPostgresDB(identity_dsn, cfg.Postgres.PgIdentityDBName)

	if err := infrastructure.RunMigrations(identity_dsn, "migrations/identity"); err != nil {
		log.Fatalf("failed to run identity migrations: %v", err)
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
		log.Printf("Identity HTTP Server running cleanly on port %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Identity HTTP Server failure: %v", err)
		}
	}()

	return srv, nil
}

func (a *IdentityApp) ShutdownHTTP(srv *http.Server) error {
	log.Println("Shutting down Identity HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced Identity HTTP shutdown failed: %w", err)
	}

	log.Println("Identity HTTP Server exited cleanly")
	return nil
}

func (a *IdentityApp) RunGRPC() (*grpc.Server, error) {

	port := a.config.Server.IdentityServiceGRPCPort
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to bind grpc tcp socket: %w", err)
	}

	srv := grpc.NewServer()

	// Create handler structure passing down needed usecases context
	grpcHandler := handler.NewIdentityGRPCHandler(a.usecases.UserUC)
	identitypb.RegisterIdentityServiceServer(srv, grpcHandler)

	go func() {
		log.Printf("Identity grpc Server running cleanly on port :%s", port)
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("Identity grpc Server failure: %v", err)
		}
	}()

	return srv, nil
}

// ShutdownGRPC cleans up and terminates the grpc processes gracefully
func (a *IdentityApp) ShutdownGRPC(srv *grpc.Server) error {
	defer a.db.Close()

	log.Println("Shutting down Identity grpc server...")

	// GracefulStop waits for all active connections and RPC requests to finish processing
	srv.GracefulStop()

	log.Println("Identity grpc Server exited cleanly")
	return nil
}
