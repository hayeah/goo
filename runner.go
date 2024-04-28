package goo

import (
	"log"

	"github.com/alexflint/go-arg"
)

type Runner[Arg any] interface {
	Run(arg *Arg) error
}

func Run[T Runner[Arg], Arg any](init func() (T, error), args *Arg) error {
	r, err := init()
	if err != nil {
		return err
	}

	err = arg.Parse(args)
	if err != nil {
		return err
	}

	err = r.Run(args)
	if err != nil {
		return err
	}

	GracefulExit()

	return nil
}

func Main[T Runner[Arg], Arg any](init func() (T, error), args *Arg) {
	err := Run(init, args)
	if err != nil {
		log.Fatalln(err)
	}
}
