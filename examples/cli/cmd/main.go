package main

import (
	"github.com/hayeah/goo/examples/cli"
)

func main() {
	mainfn, err := cli.InitMain()

	if err != nil {
		panic(err)
	}

	mainfn()
}
