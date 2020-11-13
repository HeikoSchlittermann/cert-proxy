package program

import (
	"os"
	"path/filepath"
)

var (
    Version string = `*unknown*`
    Name string = filepath.Base(os.Args[0])
    Path string = func() string {
        if p, err := filepath.Abs(os.Args[0]); err != nil {
            panic(err)
        } else {
            return p
        }
    }()
)
