//go:build wireinject

package cli

import "github.com/google/wire"

func InitApp() (*App, error) {
	panic(wire.Build(wires))
}
