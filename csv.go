package goo

import (
	"io"
	"os"

	"github.com/gocarina/gocsv"
)

// EncodeCSVFile writes a given list in CSV format
func EncodeCSV(w io.Writer, list interface{}) error {
	err := gocsv.Marshal(list, w)
	if err != nil {
		return err
	}

	return nil
}

// EncodeCSVFile writes a given list in CSV format
func EncodeCSVFile(file string, list interface{}) error {
	f, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	return EncodeCSV(f, list)
}
