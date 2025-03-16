package cli

import (
	"fmt"
	"log"

	"github.com/hayeah/goo"
	"github.com/jmoiron/sqlx"

	_ "github.com/mattn/go-sqlite3" // Import SQLite driver
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
	Migrator *goo.DBMigrator
}

func (app *App) Run() error {
	err := app.Migrator.Up([]goo.Migration{
		{
			Name: "create_users_table",
			Up: `
				CREATE TABLE users (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					email TEXT NOT NULL UNIQUE
				);
			`,
		},
	})

	if err != nil {
		return err
	}

	args := app.Args

	switch {
	case args.Checkout != nil:
		log.Printf("checkout %v", args.Checkout)
	case args.Commit != nil:
		log.Printf("commit %v", args.Commit)
	case args.Push != nil:
		log.Printf("push %v", args.Push)
	default:
		return fmt.Errorf("unknown command")
	}

	return nil
}
