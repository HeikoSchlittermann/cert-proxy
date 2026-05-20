// Copyright 2019-2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"errors"
	"testing"
)

// FuzzValidateClientName checks invariants that must hold for every
// input, not just the seeds. The asserted properties are orthogonal
// to the validator's implementation, so a bug in ValidateClientName
// can be caught without restating the validator inside the test.
//
// Properties enforced for every input:
//  1. No panic.
//  2. Every failure wraps ErrInvalidName.
//  3. ValidateClientName is strictly stricter than ValidateDomain:
//     anything ValidateClientName accepts, ValidateDomain must also
//     accept. (The converse is allowed: ValidateDomain takes
//     wildcards and Windows reserved labels that ValidateClientName
//     refuses.)
//
// The "error message never echoes the input" property is enforced by
// the handcrafted behavioural tests in cmd/cert-proxy-server, where
// the inputs resemble realistic attacker payloads. A substring-based
// check is too coarse for fuzz: trivial inputs like " " appear in
// every "contains invalid characters" message by accident.
func FuzzValidateClientName(f *testing.F) {
	seeds := []string{
		"",
		"myhost",
		"test-client",
		"a.b.c",
		"x",
		"*",
		"*.example.com",
		"sub/inner",
		`sub\inner`,
		"with\x00nul",
		".hidden",
		"..",
		"...",
		"name.",
		".name",
		"CON",
		"con",
		"CON.txt",
		"NUL",
		"COM1",
		"LPT9",
		"con.example.com",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		err := ValidateClientName(name)
		if err != nil {
			if !errors.Is(err, ErrInvalidName) {
				t.Fatalf("error from ValidateClientName(%q) does not wrap ErrInvalidName: %v", name, err)
			}

			return
		}

		if err := ValidateDomain(name); err != nil {
			t.Fatalf("ValidateClientName accepted %q but ValidateDomain rejected: %v", name, err)
		}
	})
}
