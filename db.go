package goo

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

type DatabaseConfig struct {
	Dialect string
	DSN     string
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
