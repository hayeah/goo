package goo

import (
	"errors"
)

// Skip is an error that can be returned by a function to indicate that the current map item should be skipped.
var Skip = errors.New("skip")

func Flatten[T any](collection [][]T) []T {
	result := []T{}

	for _, item := range collection {
		result = append(result, item...)
	}

	return result
}

func FlatMap[T1, T2 any](input []T1, f func(T1) ([]T2, error)) (output []T2, err error) {
	var outputs [][]T2
	for _, v := range input {
		v2, err := f(v)
		if err == Skip {
			continue
		}

		if err != nil {
			return nil, err
		}

		outputs = append(outputs, v2)
	}

	return Flatten(outputs), nil
}

func Map[T1, T2 any](input []T1, f func(T1) (T2, error)) (output []T2, err error) {
	output = make([]T2, 0, len(input))
	for _, v := range input {
		v2, err := f(v)
		if err == Skip {
			continue
		}

		if err != nil {
			return output, err
		}

		output = append(output, v2)
	}
	return output, nil
}
