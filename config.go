package goo

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
)

type Config struct {
	Database *DatabaseConfig
	Logging  *LoggerConfig
	Echo     *EchoConfig
}

func ParseArgs[T any]() (*T, error) {
	var o T

	// MustParse automatically handles help and version
	arg.MustParse(&o)
	// err := arg.Parse(&o)

	return &o, nil
}

var ErrNoConfig = fmt.Errorf("no config is found")

func ParseConfig[T any](prefix string) (*T, error) {
	prefix = strings.ToUpper(prefix)

	var o T
	// Attempt to read config as env string
	// {prefix}_CONFIG_JSON
	// {prefix}_CONFIG_TOML
	// {prefix}_CONFIG_YAML
	if prefix != "" {
		prefix = prefix + "_"
	}

	for _, format := range []string{"json", "toml", "yaml"} {
		envar := strings.ToUpper(fmt.Sprintf("%sCONFIG_%s", prefix, format))
		if envstr, ok := os.LookupEnv(envar); ok {
			// format = "json"
			err := Decode(strings.NewReader(envstr), format, &o)
			return &o, err
		}
	}

	// read as file if {prefix}_CONFIG, using file extension to determine the format:
	envar := fmt.Sprintf("%sCONFIG_FILE", prefix)
	if configFile, ok := os.LookupEnv(envar); ok {
		err := DecodeFile(configFile, &o)
		return &o, err
	}

	return nil, fmt.Errorf("%w: try setting %sCONFIG_FILE", ErrNoConfig, prefix)
}
