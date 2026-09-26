// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

// Package secret resolves credentials from URI-style sources: PASS:literal,
// FILE:/path, or ENV:VARNAME.
package secret

import (
	"fmt"
	"os"
	"strings"
)

// Read returns the secret named by src. The src is "<proto>:<value>", where
// proto is one of PASS, FILE, or ENV (case-insensitive).
func Read(src string) (string, error) {
	proto, value, ok := strings.Cut(src, `:`)
	if !ok {
		// src is not echoed: without a colon it is most likely the
		// password itself, and this error ends up in the log.
		return ``, fmt.Errorf("secret source: expected <pass|file|env>:<value> (value not shown)")
	}

	switch strings.ToUpper(proto) {
	case `PASS`:
		return value, nil
	case `FILE`:
		b, err := os.ReadFile(value)
		if err != nil {
			return ``, err
		}

		return strings.TrimRight(string(b), "\r\n \t"), nil
	case `ENV`:
		return os.Getenv(value), nil
	default:
		// Not interpolated: for "hunter2:tail" the source is the
		// password's prefix.
		return ``, fmt.Errorf("unknown secret source (expected pass, file or env)")
	}
}
