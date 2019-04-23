package cert

import (
	"errors"
	"strings"
)

// Set to satisfy flag.Value
func (format *Format) Set(value string) error {
	switch s := strings.ToUpper(value); s {
	case `PEM`:
		*format = FormatPEM
	case `PKCS12`:
		*format = FormatPKCS12
	default:
		return errors.New("Invalid format spec")
	}
	return nil
}

// String to satisfy flag.Value
func (format Format) String() string {
	return string(format)
}
