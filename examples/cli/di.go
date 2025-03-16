package cli

import (
	"github.com/google/wire"
	"github.com/hayeah/goo"
)

func ProvideConfig() (*Config, error) {
	cfg, err := goo.ParseConfig[Config]("")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func ProvideGooConfig(cfg *Config) (*goo.Config, error) {
	return &cfg.Config, nil
}

// ProvideArgs parses cli args
func ProvideArgs() (*Args, error) {
	return goo.ParseArgs[Args]()
}

func ProvideRunner() (goo.Runner, error) {
	return &App{}, nil
}

// collect all the necessary providers
var Wires = wire.NewSet(
	goo.Wires,
	// provide the base config for goo library
	ProvideGooConfig,

	// app specific providers
	ProvideConfig,
	ProvideArgs,

	// example: provide a goo.Runner interface for Main function, using a provider function
	// ProvideRunner,

	// example: provide a goo.Runner interface for Main function, by using interface binding
	wire.Struct(new(App), "*"),
	wire.Bind(new(goo.Runner), new(*App)),
)
