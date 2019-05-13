package main

import (
	"fmt"
	"strings"
)

type out string

func (o *out) Set(v string) error {
	switch s := strings.ToUpper(v); s {
	case `STDERR`, `STDOUT`:
		*o = out(s)
	default:
		return fmt.Errorf("Unknown output: %s", s)
	}
	return nil
}
func (o *out) String() string {
	return string(*o)
}
