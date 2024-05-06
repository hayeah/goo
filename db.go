package goo

import (
	"database/sql/driver"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

type DatabaseConfig struct {
	Dialect string
	DSN     string

	MigrationsPath string
}

func ProvideSQLX(goocfg *Config, down *ShutdownContext, log *zerolog.Logger) (*sqlx.DB, error) {
	if goocfg.Database == nil {
		return nil, fmt.Errorf("no database configuration")
	}

	cfg := goocfg.Database

	db, err := sqlx.Open(cfg.Dialect, cfg.DSN)
	if err != nil {
		return nil, err
	}

	down.OnExit(func() error {
		log.Debug().Str("db", cfg.DSN).Msg("closing database connection")
		return db.Close()
	})

	return db, err
}

// https://github.com/golang-migrate/migrate/blob/master/GETTING_STARTED.md
// https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md

// ProvideMigrate provides a filesystem backed db migration.
func ProvideMigrate(basecfg *Config) (*migrate.Migrate, error) {
	if basecfg.Database == nil {
		return nil, fmt.Errorf("no database configuration")
	}

	cfg := basecfg.Database

	if cfg.MigrationsPath == "" {
		return nil, fmt.Errorf("no MigrationsPath configured")
	}

	databaseURL := fmt.Sprintf("%s://%s", cfg.Dialect, cfg.DSN)
	fileURL := fmt.Sprintf("file://%s", cfg.MigrationsPath)

	m, err := migrate.New(fileURL, databaseURL)
	if err != nil {
		return nil, err
	}

	return m, nil
}

/*
The benefit of embedding is to make it easier to distribute the application as a
single binary. The embedded files are stored in the binary itself, so there is
no need to distribute the migration files separately.

// go:embed testdata/migrations/*.sql
//  var fs embed.FS

*/

type EmbbededMigrate migrate.Migrate

type EmbeddedMigrateConfig struct {
	fs        embed.FS
	embedPath string
}

// ProvideEmbbededMigrate provides an embed.FS based db migration.
func ProvideEmbbededMigrate(embedCfg *EmbeddedMigrateConfig, cfg *Config) (*EmbbededMigrate, error) {
	if cfg.Database == nil {
		return nil, fmt.Errorf("no database configuration")
	}

	fs, err := iofs.New(embedCfg.fs, embedCfg.embedPath)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithSourceInstance("iofs", fs, cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	return (*EmbbededMigrate)(m), err
}

type JSONColumn[T any] struct {
	V T
}

func (j *JSONColumn[T]) Scan(src any) error {
	if src == nil {
		return nil
	}
	return json.Unmarshal(src.([]byte), &j.V)
}

func (j *JSONColumn[T]) Value() (driver.Value, error) {
	raw, err := json.Marshal(j.V)
	return raw, err
}

// MarshalJSON
func (j JSONColumn[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.V)
}

// UnmarshalJSON
func (j *JSONColumn[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.V)
}
