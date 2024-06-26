package goo

import (
	"database/sql/driver"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
)

type DatabaseConfig struct {
	Dialect string
	DSN     string

	MigrationsPath        string
	MigrationsRunManually bool
}

func ProvideSQLX(goocfg *Config, down *ShutdownContext, log *slog.Logger) (*sqlx.DB, error) {
	if goocfg.Database == nil {
		return nil, fmt.Errorf("no database configuration")
	}

	cfg := goocfg.Database

	db, err := sqlx.Open(cfg.Dialect, cfg.DSN)
	if err != nil {
		return nil, err
	}

	down.OnExit(func() error {
		log.Debug("closing database connection", "db", cfg.DSN)
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

	databaseURL := fmt.Sprintf("%s://file:%s", cfg.Dialect, cfg.DSN)
	fileURL := fmt.Sprintf("file://%s", cfg.MigrationsPath)

	m, err := migrate.New(fileURL, databaseURL)
	if err != nil {
		return nil, err
	}

	if cfg.MigrationsRunManually {
		return m, nil
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		err = nil
	}

	return m, nil
}

/*
The benefit of embedding is to make it easier to distribute the application as a
single binary. The embedded files are stored in the binary itself, so there is
no need to distribute the migration files separately.

//go:embed testdata/migrations/*.sql
//var fs embed.FS

*/

type EmbbededMigrate migrate.Migrate

type EmbeddedMigrateConfig struct {
	FS        embed.FS
	EmbedPath string
}

// ProvideEmbbededMigrate provides an embed.FS based db migration.
func ProvideEmbbededMigrate(embedCfg *EmbeddedMigrateConfig, basecfg *Config) (*EmbbededMigrate, error) {
	if basecfg.Database == nil {
		return nil, fmt.Errorf("no database configuration")
	}

	cfg := basecfg.Database

	fs, err := iofs.New(embedCfg.FS, embedCfg.EmbedPath)
	if err != nil {
		return nil, err
	}

	dsn := cfg.DSN
	if cfg.Dialect == "sqlite3" {
		// golang-migrate parses the DNS into URL first, which causes whitespaces to
		// be escaped as `%20`. However, by default, sqlite3 does not treat the DSN
		// as proper URL.
		//
		// To work around this, add the "file:" prefix to the DSN, and sqlite3 will
		// treat it as proper URI, and "%20" will be treated as whitespace.
		//
		// https://github.com/golang-migrate/migrate/pull/500#discussion_r566560229
		//
		// They didn't actually fix this problem, but added a test case "as
		// reference", lmao.
		dsn = fmt.Sprintf("file:%s", cfg.DSN)
	}

	databaseURL := fmt.Sprintf("%s://%s", cfg.Dialect, dsn)

	m, err := migrate.NewWithSourceInstance("iofs", fs, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if !cfg.MigrationsRunManually {
		err = m.Up()
		if err == migrate.ErrNoChange {
			err = nil
		}

		if err != nil {
			return nil, fmt.Errorf("migrate with embed: %w", err)
		}
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

	switch src := src.(type) {
	case []byte:
		return json.Unmarshal(src, &j.V)
	case string:
		return json.Unmarshal([]byte(src), &j.V)
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
}

func (j *JSONColumn[T]) Value() (driver.Value, error) {
	raw, err := json.Marshal(j.V)
	if err != nil {
		return nil, err
	}
	return string(raw), err
}

// MarshalJSON
func (j JSONColumn[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.V)
}

// UnmarshalJSON
func (j *JSONColumn[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.V)
}

// TimeColumn stores time.Time as int64 in SQLITE
type TimeColumn struct {
	time.Time
}

// Scan implements the Scanner interface for JSONDateTime
// Expects time as uint64 from the database.
func (jdt *TimeColumn) Scan(src any) error {
	if src == nil {
		return nil
	}

	var unixTime uint64
	switch src := src.(type) {
	case int64:
		unixTime = uint64(src)
	case uint64:
		unixTime = src
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}

	jdt.Time = time.UnixMilli(int64(unixTime))
	return nil
}

// Value implements the Valuer interface for JSONDateTime
// Returns the time as uint64 UNIX timestamp.
func (jdt TimeColumn) Value() (driver.Value, error) {
	return jdt.UnixMilli(), nil
}

// MarshalJSON converts the JSONDateTime to a JSON string in ISO format.
func (jdt TimeColumn) MarshalJSON() ([]byte, error) {
	return json.Marshal(jdt.Time.Format(time.RFC3339))
}

// UnmarshalJSON parses an ISO format JSON string into JSONDateTime.
func (jdt *TimeColumn) UnmarshalJSON(data []byte) error {
	var isoStr string
	if err := json.Unmarshal(data, &isoStr); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339, isoStr)
	if err != nil {
		return err
	}
	jdt.Time = t
	return nil
}
