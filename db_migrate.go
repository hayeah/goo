package goo

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

// Migration represents a single database migration
type Migration struct {
	Name string
	Up   string
}

// DBMigrator handles database migrations using sqlx.DB
type DBMigrator struct {
	db     *sqlx.DB
	logger *slog.Logger
}

// ProvideDBMigrator creates a new Migrator instance
func ProvideDBMigrator(db *sqlx.DB, logger *slog.Logger) *DBMigrator {
	return &DBMigrator{
		db:     db,
		logger: logger,
	}
}

// Up runs all pending migrations
func (m *DBMigrator) Up(migrations []Migration) error {
	// Create a table to track which migrations have been applied
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of already-applied migrations
	rows, err := m.db.Queryx("SELECT name FROM migrations;")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	// Build a set of applied migrations
	appliedMigrations := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("failed to scan migration name: %w", err)
		}
		appliedMigrations[name] = true
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating migration rows: %w", err)
	}

	// Apply each migration that hasn't been applied yet
	for _, migration := range migrations {
		if !appliedMigrations[migration.Name] {
			m.logger.Info("Applying migration", "name", migration.Name)

			// Execute the migration within a transaction
			tx, err := m.db.Beginx()
			if err != nil {
				return fmt.Errorf("failed to begin transaction for migration %s: %w", migration.Name, err)
			}

			// Apply the migration
			_, err = tx.Exec(migration.Up)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to apply migration %s: %w", migration.Name, err)
			}

			// Mark the migration as applied
			_, err = tx.Exec(
				"INSERT INTO migrations (name, applied_at) VALUES (?, ?)",
				migration.Name,
				time.Now(),
			)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
			}

			// Commit the transaction
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit migration %s: %w", migration.Name, err)
			}
		}
	}

	return nil
}
