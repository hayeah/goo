//go:build wireinject

package cli

import (
	"github.com/google/wire"
	"github.com/hayeah/goo"
)

func InitMain() (goo.Main, error) {
	panic(wire.Build(Wires))
}
