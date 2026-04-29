// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	"os"
	"testing"
)

func TestRead(t *testing.T) {
	// String
	t.Log("PASS")

	if pass, err := Read("PASS:foo"); err != nil {
		t.Errorf("unexpected: %v\n", err)
	} else if pass != "foo" {
		t.Errorf("expected %q, got %q\n", "foo", pass)
	}

	// Environment
	t.Log("ENV")

	_ = os.Setenv("PW", "bar")

	if pass, err := Read("ENV:PW"); err != nil {
		t.Errorf("unexpected: %v\n", err)
	} else if pass != "bar" {
		t.Errorf("expected %q, got %q\n", "bar", pass)
	}

	// File
	t.Log("FILE")

	if pass, err := Read("FILE:pwfile"); err != nil {
		t.Errorf("unexpected: %v\n", err)
	} else if pass != "baz" {
		t.Errorf("expected %q, got %q\n", "baz", pass)
	}
}
