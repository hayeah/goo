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

// ParseArgs parses command-line arguments into a struct of type T.
// It uses the github.com/alexflint/go-arg package to automatically handle
// parsing, help text generation, and version information.
//
// The struct type T should be annotated with `arg` tags to define how
// command-line arguments map to struct fields.
//
// Example:
//
//	type Args struct {
//	    Verbose bool   `arg:"-v,--verbose" help:"enable verbose logging"`
//	    Port    int    `arg:"-p" help:"port number"`
//	    Files   []string `arg:"positional"`
//	}
//
//	args, err := goo.ParseArgs[Args]()
//
// Returns a pointer to the populated struct and nil error on success.
// Note that this function uses MustParse internally, which will exit the
// program if parsing fails or if --help or --version flags are provided.
func ParseArgs[T any]() (*T, error) {
	var o T

	// MustParse automatically handles help and version
	arg.MustParse(&o)
	// err := arg.Parse(&o)

	return &o, nil
}

var ErrNoConfig = fmt.Errorf("no config is found")

// ParseConfig loads configuration from environment variables into a struct of type T.
// It attempts to find configuration in the following order:
//
// 1. Environment variables containing configuration data in different formats:
//   - {prefix}_CONFIG_JSON: JSON-formatted configuration string
//   - {prefix}_CONFIG_TOML: TOML-formatted configuration string
//   - {prefix}_CONFIG_YAML: YAML-formatted configuration string
//
// 2. Environment variable pointing to a configuration file:
//   - {prefix}_CONFIG_FILE: Path to a configuration file (format determined by file extension)
//
// The prefix parameter is used to namespace environment variables and can be empty.
// If provided, it will be automatically converted to uppercase and appended with "_".
// If prefix is an empty string, the environment variables will be simply CONFIG_JSON,
// CONFIG_TOML, CONFIG_YAML, or CONFIG_FILE without any prefix.
//
// Example:
//
//	type Config struct {
//	    Host    string `json:"host" yaml:"host" toml:"host"`
//	    Port    int    `json:"port" yaml:"port" toml:"port"`
//	    LogFile string `json:"log_file" yaml:"log_file" toml:"log_file"`
//	}
//
//	// Will look for APP_CONFIG_JSON, APP_CONFIG_TOML, APP_CONFIG_YAML, or APP_CONFIG_FILE
//	config, err := goo.ParseConfig[Config]("APP")
//
//	// Will look for CONFIG_JSON, CONFIG_TOML, CONFIG_YAML, or CONFIG_FILE
//	defaultConfig, err := goo.ParseConfig[Config]("")
//
// Returns a pointer to the populated struct on success, or an error if no configuration
// source is found or if parsing fails. If no configuration is found, returns ErrNoConfig
// with a suggestion to set the appropriate environment variable.
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
