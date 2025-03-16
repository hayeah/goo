package goo

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

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
