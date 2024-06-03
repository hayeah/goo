package goo

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
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
func TypedLogger(log *slog.Logger, srv interface{}) *slog.Logger {
	return log.With("_type", InterfaceName(srv))
}

type LoggerConfig struct {
	LogLevel  string
	LogFile   string
	LogFormat string // json, console
}

func ProvideSlog(cfg *Config) (*slog.Logger, error) {
	lvlText := strings.TrimSpace(strings.ToUpper(cfg.Logging.LogLevel))
	if lvlText == "" {
		lvlText = "INFO"
	}

	var level slog.Level
	err := level.UnmarshalText([]byte(lvlText))
	if err != nil {
		return nil, fmt.Errorf("provide slog: %w", err)
	}

	var handler slog.Handler
	handlerOptions := &slog.HandlerOptions{Level: level}

	switch cfg.Logging.LogFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, handlerOptions)
	default:
		handler = slog.NewTextHandler(os.Stderr, handlerOptions)
	}

	log := slog.New(handler)

	return log, nil
}
