package cli

import (
	"fmt"
	"log"

	"github.com/hayeah/goo"
	"github.com/jmoiron/sqlx"
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
	Args     *Args
	Config   *Config
	Shutdown *goo.ShutdownContext
	DB       *sqlx.DB
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
