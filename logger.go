package goo

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/rs/zerolog"

	"github.com/rs/zerolog/log"
)

type Named interface {
	Name() string
}

// InterfaceName returns the type name of the interface.
// If it implements Named, it will return the name from that interface.
func InterfaceName(o interface{}) string {
	if n, ok := o.(Named); ok {
		return n.Name()
	} else {
		return reflect.TypeOf(o).Elem().Name()
	}
}

// TypedLogger returns a logger that adds "_type" to its logging context
func TypedLogger(log *zerolog.Logger, srv interface{}) *zerolog.Logger {
	pdlog := log.With().Str("_type", InterfaceName(srv)).Logger()
	return &pdlog
}

type LoggerConfig struct {
	LogLevel  string
	LogFile   string
	LogFormat string // json, console
}

func ProvideZeroLogger(goocfg *Config, shutdown *ShutdownContext) (*zerolog.Logger, error) {
	if goocfg.Logging == nil {
		return nil, fmt.Errorf("no logging configuration")
	}

	cfg := goocfg.Logging

	// configure timestamp etc...
	log := log.Logger

	if cfg.LogLevel == "" {
		log = log.Level(zerolog.InfoLevel)
	} else {
		level, err := zerolog.ParseLevel(cfg.LogLevel)
		if err != nil {
			return nil, err
		}

		log = log.Level(level)
	}

	var out io.Writer = os.Stderr

	if cfg.LogFile != "" {
		file, err := os.OpenFile(
			cfg.LogFile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0664,
		)

		if err != nil {
			return nil, err
		}

		out = file

		shutdown.OnExit(func() error {
			return file.Close()
		})
	}

	switch cfg.LogFormat {
	case "", "json":
		log = log.Output(out)
	case "console":
		log = log.Output(zerolog.ConsoleWriter{Out: out})
	default:
		return nil, fmt.Errorf("unsupported log format: %s", cfg.LogFormat)
	}

	shutdown.logger = &log

	return &log, nil
}
