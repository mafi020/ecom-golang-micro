package infrastructure

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		// Stop scanning if we discover the main project go.mod configuration file
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached file system root boundary
		}
		dir = parent
	}
	return "."
}

func RunMigrations(dsn, path string) error {
	projectRoot := findProjectRoot()
	migrationsPath := filepath.Join(projectRoot, path)

	log.Printf("Initializing OS File System Driver for path: %s", migrationsPath)

	// 1. Create a native file driver instance pointing directly to your SQL folder
	fileDriver, err := (&file.File{}).Open(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to initialize migration file driver: %w", err)
	}

	// 2. Pass the driver object instance directly into the migrator engine
	m, err := migrate.NewWithSourceInstance(
		"file",
		fileDriver,
		dsn,
	)
	if err != nil {
		return fmt.Errorf("failed to build migrator instance: %w", err)
	}
	defer m.Close()

	// 3. Execute migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration execution failed: %w", err)
	}

	log.Println("Database migrations synchronized and applied successfully!")
	return nil
}
