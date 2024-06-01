package goo

import "github.com/google/wire"

var Wires = wire.NewSet(
	ProvideShutdownContext,
	ProvideZeroLogger,
	ProvideEcho,
	ProvideSQLX,
	ProvideMigrate,
	ProvideEmbbededMigrate,
)
