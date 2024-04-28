package goo

import (
	"fmt"
	"os"
	"strings"
)

func DecodeConfig[T any](prefix string) (*T, error) {
	prefix = strings.ToUpper(prefix)

	var o T
	// Attempt to read config as env string
	// {prefix}_CONFIG_JSON
	// {prefix}_CONFIG_TOML
	// {prefix}_CONFIG_YAML
	for _, format := range []string{"json", "toml", "yaml"} {
		envar := strings.ToUpper(fmt.Sprintf("%s_CONFIG_%s", prefix, format))
		if envstr, ok := os.LookupEnv(envar); ok {
			// format = "json"
			err := Decode(strings.NewReader(envstr), format, &o)
			return &o, err
		}
	}

	// read as file if {prefix}_CONFIG, using file extension to determine the format:
	envar := fmt.Sprintf("%s_CONFIG_FILE", prefix)
	if configFile, ok := os.LookupEnv(envar); ok {
		err := DecodeFile(configFile, &o)
		return &o, err
	}

	return nil, fmt.Errorf("no config is found. Try setting %s_CONFIG_FILE", prefix)
}

// LoadConfig decodes {prefix}_CONFIG as config file, or {prefix}_CONFIG_JSON
func LoadConfig(prefix string, o interface{}) error {
	// read as JSON data if {prefix}_CONFIG_JSON
	configJSONEnv := fmt.Sprintf("%s_CONFIG_JSON", prefix)
	configJSONString, ok := os.LookupEnv(configJSONEnv)
	if ok {
		return Decode(strings.NewReader(configJSONString), "json", &o)
	}

	// read as JSON data if {prefix}_CONFIG_TOML
	configTOMLEnv := fmt.Sprintf("%s_CONFIG_TOML", prefix)
	configTOMLString, ok := os.LookupEnv(configTOMLEnv)
	if ok {
		return Decode(strings.NewReader(configTOMLString), "toml", &o)
	}

	// read as JSON data if {prefix}_CONFIG_YAML
	configYAMLEnv := fmt.Sprintf("%s_CONFIG_YAML", prefix)
	configYAMLString, ok := os.LookupEnv(configYAMLEnv)
	if ok {
		return Decode(strings.NewReader(configYAMLString), "yaml", &o)
	}

	// read as file if {prefix}_CONFIG
	// using file extension to determine the format:
	configFileVar := fmt.Sprintf("%s_CONFIG", prefix)
	configFile, ok := os.LookupEnv(configFileVar)
	if !ok {
		return fmt.Errorf("expects env to be set: %s", configFileVar)
	}

	return DecodeFile(configFile, &o)

}
