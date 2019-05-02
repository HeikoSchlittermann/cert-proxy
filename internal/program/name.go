package program

import (
	"os"
	"path/filepath"
)

var Name string

func init() {
	var err error
	Name, err = filepath.Abs(os.Args[0])
	if err != nil {
		panic(err)
	}
}
