package main

import (
	"github.com/hayeah/goo/examples/cli"
)

func main() {
	fn, err := cli.InitMain()

	if err != nil {
		panic(err)
	}

	fn()
}
