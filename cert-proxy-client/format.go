package main

import (
	"errors"
	"strings"
)

const (
	FormatInvalid = iota // we can't 0 this as the default in the flag package
	FormatPEM
	FormatPKCS12
)

type Format int

var ITEMS = map[Format][]string{
	FormatPEM:    []string{"cert", "chain", "fullchain", "privkey"},
	FormatPKCS12: []string{"pkcs12"},
}

// Implement the flag.Value interface
// interface {
// - Set(string) error
// - String() string
func (format *Format) Set(value string) (err error) {
	switch s := strings.ToUpper(value); s {
	case "PEM":
		*format = FormatPEM
	case "PKCS12":
		*format = FormatPKCS12
	default:
		return errors.New("Wrong format spec")
	}
	return nil
}

func (format Format) String() string {
	switch format {
	case FormatPEM:
		return "PEM"
	case FormatPKCS12:
		return "PKCS12"
	default:
		return "<invalid format>"
	}
}
