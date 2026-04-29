// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

// Package secret resolves credentials from URI-style sources: PASS:literal,
// FILE:/path, or ENV:VARNAME.
package secret

import (
	"io/ioutil"
	"os"
	"strings"
)

// Read returns the secret named by src. The src is "<proto>:<value>", where
// proto is one of PASS, FILE, or ENV (case-insensitive).
func Read(src string) (string, error) {
	var proto, value = func() (string, string) {
		x := strings.SplitN(src, `:`, 2)
		return x[0], x[1]
	}()

	switch strings.ToUpper(proto) {
	case `PASS`:
		return value, nil
	case `FILE`:
		b, err := ioutil.ReadFile(value)
		if err != nil {
			return ``, err
		}

		return strings.TrimRight(string(b), "\r\n \t"), nil
	case `ENV`:
		return os.Getenv(value), nil
	default:
		panic("unhandled secret source proto: " + proto)
	}
}
