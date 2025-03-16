package goo

import "github.com/google/wire"

var Wires = wire.NewSet(
	ProvideShutdownContext,
	ProvideSlog,
	ProvideEcho,
	ProvideSQLX,
	ProvideMain,
)
