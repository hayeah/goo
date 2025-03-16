package main

import (
	"github.com/hayeah/goo-examples/cli"
)

func main() {
	app, err := cli.InitApp()
	if err != nil {
		panic(err)
	}

	err = app.Run()
	if err != nil {
		panic(err)
	}
}
