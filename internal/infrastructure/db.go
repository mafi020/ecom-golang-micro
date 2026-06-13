package infrastructure

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// NewPostgresDB creates a new PostgreSQL connection pool.
func NewPostgresDB(dsn, dbName string) *sql.DB {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		slog.Error("CRITICAL: Invalid database connection pool parameters string", "err", err)
		return nil
	}

	// Configure default pool connection limits
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 🚀 THE CURE: Spin up a non-blocking background connection loop tracker
	go func() {
		for {
			// Trigger a non-blocking ping handshake verification check
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := db.PingContext(ctx)
			cancel()

			if err == nil {
				log.Printf("🎉 %s DATABASE CONNECTION SUCCESSFULLY ESTABLISHED AND ACTIVE", dbName)
				break // Handshake succeeded! Exit loop and continue tracking context
			}

			// 🚨 Database is down: Log it but DO NOT call log.Fatalf or os.Exit
			slog.Warn("⚠️ DATABASE IS UNREACHABLE. Microservice remains alive. Retrying connection in 5 seconds...",
				"err", err,
				"dsn_lookup", dsn,
			)

			// Wait 5 seconds before attempting another database link verification
			time.Sleep(5 * time.Second)
		}
	}()

	return db
}
