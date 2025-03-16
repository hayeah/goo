package cli

import (
	"fmt"
	"log"

	"github.com/google/wire"
	"github.com/hayeah/goo"
)

type Config struct {
	goo.Config
	OpenAI OpenAIConfig
}

type OpenAIConfig struct {
	APIKey string
}

type CheckoutCmd struct {
	Branch string `arg:"positional"`
	Track  bool   `arg:"-t"`
}

type CommitCmd struct {
	All     bool   `arg:"-a"`
	Message string `arg:"-m"`
}

type PushCmd struct {
	Remote      string `arg:"positional"`
	Branch      string `arg:"positional"`
	SetUpstream bool   `arg:"-u"`
}

type Args struct {
	Checkout *CheckoutCmd `arg:"subcommand:checkout"`
	Commit   *CommitCmd   `arg:"subcommand:commit"`
	Push     *PushCmd     `arg:"subcommand:push"`
}

type App struct {
	Args   *Args
	Config *Config
}

func (a *App) Run() error {
	switch {
	case a.Args.Checkout != nil:
		log.Printf("checkout %v", a.Args.Checkout)
	case a.Args.Commit != nil:
		log.Printf("commit %v", a.Args.Commit)
	case a.Args.Push != nil:
		log.Printf("push %v", a.Args.Push)
	default:
		return fmt.Errorf("unknown command")
	}

	return nil
}

// ProvideConfig loads the configuration from the environment.
func ProvideConfig() (*Config, error) {
	return goo.ParseConfig[Config]("")
}

// ProvideArgs parses cli args
func ProvideArgs() (*Args, error) {
	return goo.ParseArgs[Args]()
}

func ProvideGooConfig(cfg *Config) *goo.Config {
	return &cfg.Config
}

var wires = wire.NewSet(
	goo.Wires,

	ProvideGooConfig,
	ProvideConfig,
	ProvideArgs,

	wire.Struct(new(App), "*"),
)
