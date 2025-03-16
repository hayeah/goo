Below is a set of instructions and conventions for building a “goo”-based application. This illustrates how to bring together **config**, **logging**, **database** connections, and **graceful shutdown** via `goo`’s helpers. The code references below come directly from this repository’s structure and from the sample `examples/cli` app.

---

## 1. Project Layout Convention

A common structure using goo might look like this:

```
yourapp/
├── di.go               // Wire providers: glue everything together.
├── app.go              // The main application struct implementing goo.Runner.
├── cmd/main.go             // The actual main() entrypoint. Minimal – calls InitMain().
├── cfg.toml            // Example config file (TOML, JSON, YAML, etc.).
├── wire.go             // The wire.Build(...) definitions.
├── wire_gen.go         // Generated by wire. DO NOT EDIT.
├── Makefile            // Typical build steps, including “wire” code generation, “run”, “dev”.
└── ...
```

### High-Level Flow

1. **`main.go`** calls `InitMain()`, gets a `goo.Main` function, and calls that function.  
2. **`InitMain()`** is defined in `wire.go` using Google’s Wire for dependency injection.  
3. **`di.go`** has a `Wires = wire.NewSet(...)` that enumerates all providers from the `goo` library (`goo.Wires`) plus your own custom providers.  
4. The `App` struct in `app.go` implements `goo.Runner`. Inside `App.Run()`, you can do any final initialization such as database migrations.  

## 2. Defining Your Config

`goo.Config` is the baseline config for databases, logging, and echo server. You can embed it inside your own config struct if you want to add additional fields. For example:

```go
// In app.go (or in a dedicated config.go):
package cli

import (
    "github.com/hayeah/goo"
)

type Config struct {
    goo.Config           // embed goo’s config
    OpenAI struct {
        APIKey string
    }
    // ... other fields
}
```

### How `goo.Config` Loads

`goo.Config` is populated via environment variables or a config file. For example:

- `CONFIG_FILE=cfg.toml go run ./cmd`

Inside `goo.ParseConfig[prefix]`, goo will look for:

- `prefix_CONFIG_JSON`  
- `prefix_CONFIG_TOML`  
- `prefix_CONFIG_YAML`  
- `prefix_CONFIG_FILE`  

… If `prefix` is empty, it defaults to `CONFIG_JSON`, `CONFIG_TOML`, etc. Once the file or environment string is found, `goo` decodes it into your struct.

## 3. The `App` Struct (Implements `goo.Runner`)

The application struct must have a `Run() error` method. This is typically where you handle your main logic and do any top-level tasks.

Example (from `examples/cli/app.go`):

```go
type App struct {
    Args     *Args              // For parsed CLI arguments
    Config   *Config            // For loaded config
    Shutdown *goo.ShutdownContext
    DB       *sqlx.DB
    Migrator *goo.DBMigrator
}

func (app *App) Run() error {
    // Example: run migrations
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
        // ... more migrations
    })
    if err != nil {
        return err
    }

    // Then do the actual logic, e.g. handle CLI subcommands
    switch {
    case app.Args.Checkout != nil:
        // ...
    case app.Args.Commit != nil:
        // ...
    default:
        return fmt.Errorf("unknown command")
    }

    return nil
}
```

`App`’s fields are wired (injected) by the providers described below.

## 4. Using Wire for DI

### `di.go`

You typically collect all providers in a `Wires` variable:

```go
package cli

import (
    "github.com/google/wire"
    "github.com/hayeah/goo"
)

func ProvideConfig() (*Config, error) {
    // Parse environment + file
    cfg, err := goo.ParseConfig[Config]("")  // prefix = ""
    if err != nil {
        return nil, err
    }

    return cfg, nil
}

func ProvideGooConfig(cfg *Config) (*goo.Config, error) {
    // The goo library specifically needs a pointer to the embedded goo.Config
    return &cfg.Config, nil
}

func ProvideArgs() (*Args, error) {
    // CLI argument parsing
    return goo.ParseArgs[Args]()
}

var Wires = wire.NewSet(
    goo.Wires,          // includes ProvideShutdownContext, ProvideSlog, ProvideSQLX, ProvideDBMigrator, ProvideMain
    ProvideGooConfig,   // turn your *Config into *goo.Config
    ProvideConfig,
    ProvideArgs,
    // Provide your own App
    wire.Struct(new(App), "*"),
    wire.Bind(new(goo.Runner), new(*App)),
)
```

### `wire.go`

```go
//go:build wireinject

package cli

import (
    "github.com/google/wire"
    "github.com/hayeah/goo"
)

func InitMain() (goo.Main, error) {
    panic(wire.Build(Wires))
}
```

### `wire_gen.go`

After running `wire`, a generated file `wire_gen.go` will appear with the actual “glue” code.  
Check your Makefile for a rule that may look like:

```Makefile
.PHONY: wire
wire:
    go run github.com/google/wire/cmd/wire .
```

## 5. The `main.go`

Finally, in `main.go`, you just call your `InitMain()` to get a `goo.Main` function, and invoke it:

```go
package main

import (
    "github.com/hayeah/goo/examples/cli"
)

func main() {
    mainfn, err := cli.InitMain()
    if err != nil {
        panic(err)
    }
    mainfn() // calls goo.Main, which will run your App.Run and do graceful shutdown
}
```

`goo.Main` is itself a `func()` that orchestrates your application logic (`runner.Run()`) and calls `shutdown.doExit(...)` if any error occurs.

## 6. Graceful Shutdown

`goo.ShutdownContext`:

- Watches for SIGINT signals.
- Allows you to register cleanup routines (`OnExit(func() error)`) that run before exit.
- Offers `BlockExit(func() error)` if you need to do an operation that must finish before the app terminates.

`goo.ProvideShutdownContext` will be automatically included if you import `goo.Wires`.  
As soon as your app calls `mainfn()`, `ProvideMain(...)` ensures that on any error from your `App.Run()`, it calls `doExit(...)` gracefully.

### Examples

#### Registering Cleanup Functions

```go
func (a *App) Run() error {
    // Get the shutdown context from DI
    shutdown := a.shutdown
    
    // Register a cleanup function to be executed during shutdown
    shutdown.OnExit(func() error {
        a.logger.Info("closing database connection")
        return a.db.Close()
    })
    
    // Register another cleanup function
    shutdown.OnExit(func() error {
        a.logger.Info("cleaning up temporary files")
        return os.RemoveAll(a.tempDir)
    })
    
    // ... rest of your application logic
    return nil
}
```

#### Ensuring Operations Complete Before Shutdown

```go
func (a *App) ProcessImportantData() error {
    // This operation must complete before the application terminates
    return a.shutdown.BlockExit(func() error {
        a.logger.Info("processing important data")
        
        // Simulate a long-running operation
        for i := 0; i < 100; i++ {
            // Check if we should abort
            select {
            case <-a.shutdown.Done():
                a.logger.Warn("shutdown requested, finishing critical work")
                // Complete essential work quickly
                return nil
            default:
                // Continue normal processing
                time.Sleep(100 * time.Millisecond)
            }
        }
        
        a.logger.Info("important data processing completed")
        return nil
    })
}
```

## 7. Database Setup & Migrations

If your config includes a `DatabaseConfig` (as part of `goo.Config`), you can rely on:

- `goo.ProvideSQLX` – opens `sqlx.DB`.
- `goo.ProvideDBMigrator` – returns a migrator.  
- `App.Run()` can call `Migrator.Up(...)` with your table definitions or direct SQL migrations.

### Example Database Config Snippet (`cfg.toml`)

```toml
[Database]
Dialect = "sqlite3"
DSN = "mydb.sqlite3"
MigrationsPath = "./migrations"

[Logging]
LogLevel = "DEBUG"

[Echo]
Listen = ":8000"
```

## 8. Logging

`goo.ProvideSlog(cfg *Config)` sets up a `slog.Logger` with a specified level, format, etc.  
If you embed `Logging` inside your config, you’ll get these fields:

```toml
[Logging]
LogLevel = "DEBUG"    # or INFO, ERROR, etc.
LogFormat = "json"    # or leave empty for console
```

The `slog.Logger` is then used everywhere you see `logger *slog.Logger`.

## 9. Echo Server (Optional)

If you want an HTTP server, `goo.ProvideEcho` sets one up with logging, error handlers, CORS, etc. Your `Config.Echo.Listen` can specify the listening address. Typically, you’d do something like:

```go
e := goo.ProvideEcho(logger)
e.GET("/ping", func(c echo.Context) error {
    return c.String(http.StatusOK, "pong")
})
go e.Start(cfg.Echo.Listen)
```

(Though you can integrate it further with `wire` and your `App`.)

## 10. Putting It All Together

1. **Create your config struct** with `goo.Config` embedded, plus any extra fields.  
2. **Define `App`** that implements `goo.Runner`.  
3. **Write providers** in `di.go` that produce your `*Config`, `*goo.Config`, plus your `App`.  
4. **Create a `Wires` set** that includes `goo.Wires` and your custom providers.  
5. **Add `wire.Build(Wires)`** in `wire.go` in an `InitMain()` function that returns `(goo.Main, error)`.  
6. **Generate code** via `make wire`.  
7. **Use that `InitMain()`** in `main.go`:

   ```go
   func main() {
     mainfn, err := InitMain()
     if err != nil {
       panic(err)
     }
     mainfn()
   }
   ```
8. **Run** via `make run` (or `go run .` if you set env vars properly).  


## Example Makefile

```
.PHONY: wire run dev

wire:
	go run github.com/google/wire/cmd/wire .

# make run ARGS="somefile.go --flag1 --flag2"
run:
	CONFIG_FILE=cfg.toml go run ./cmd $(ARGS)

dev:
	CONFIG_FILE=cfg.toml go run github.com/cortesi/modd/cmd/modd@latest
```
