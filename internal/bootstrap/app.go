package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang/config"
	utils_http "github.com/mafi020/ecom-golang/internal/delivery/http/utils"
	"github.com/mafi020/ecom-golang/internal/infrastructure"
)

type App struct {
	db       *sql.DB
	config   *config.Config
	usecases *Usecases // Cached usecases shared between HTTP and gRPC
}

func InitializeApp() *App {
	cfg := config.LoadConfig()
	db := initDatabase(cfg)

	transactor := infrastructure.NewPostgresTransactor(db)
	repos := RegisterRepositories(db)
	usecases := RegisterUsecases(repos, transactor, cfg)

	// Save usecases context to the root application struct container
	return &App{db: db, config: cfg, usecases: usecases}
}

func initDatabase(cfg *config.Config) *sql.DB {
	db := infrastructure.NewPostgresDB(cfg.PostgresDSN())

	if err := infrastructure.RunMigrations(cfg.PostgresDSN()); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func initRouter() *gin.Engine {
	utils_http.SetupValidator()

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(cors.Default())
	r.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		log.Printf("panic recovered: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
	}))

	return r
}

// ── HTTP RUNTIME LIFECYCLE ───────────────────────────────────────────────────

func (a *App) RunHTTP() (*http.Server, error) {
	engine := initRouter()
	RegisterHTTPHandlers(engine, a.usecases, a.config)
	srv := &http.Server{
		Addr:    ":" + a.config.Server.MonolithHTTPPort,
		Handler: engine,
	}

	go func() {
		log.Printf("HTTP Server starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP Server error: %v", err)
		}
	}()

	return srv, nil
}

func (a *App) ShutdownHTTP(srv *http.Server) error {
	defer a.db.Close()

	log.Println("Shutting down HTTP server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced HTTP shutdown failed: %w", err)
	}

	log.Println("HTTP Server exited cleanly")
	return nil
}
