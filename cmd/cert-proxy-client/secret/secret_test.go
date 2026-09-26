// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	"os"
	"strings"
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

func TestReadTable(t *testing.T) {
	t.Setenv("CERT_PROXY_SECRET_TEST", "fromenv")

	tests := []struct {
		src, want, wantErr string
	}{
		{src: "pass:foo", want: "foo"},
		{src: "PASS:foo:bar", want: "foo:bar"},
		{src: "pass:", want: ""},
		{src: "env:CERT_PROXY_SECRET_TEST", want: "fromenv"},
		{src: "file:pwfile", want: "baz"},
		{src: "file:does-not-exist", wantErr: "does-not-exist"},
		{src: "hunter2", wantErr: `secret source: expected <pass|file|env>:<value> (value not shown)`},
		{src: "", wantErr: `secret source: expected <pass|file|env>:<value> (value not shown)`},
		{src: "bogus:x", wantErr: `unknown secret source (expected pass, file or env)`},
		{src: ":x", wantErr: `unknown secret source (expected pass, file or env)`},
		{src: "hunter2:tail", wantErr: `unknown secret source (expected pass, file or env)`},
	}

	for _, tc := range tests {
		t.Run(tc.src, func(t *testing.T) {
			got, err := Read(tc.src)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Read(%q) error = %v, want %q", tc.src, err, tc.wantErr)
				}

				if !strings.Contains(tc.src, ":") && tc.src != "" && strings.Contains(err.Error(), tc.src) {
					t.Errorf("Read(%q): error echoes the (likely) password: %v", tc.src, err)
				}

				if strings.Contains(err.Error(), "hunter2") {
					t.Errorf("Read(%q): error echoes the password prefix: %v", tc.src, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("Read(%q): unexpected error %v", tc.src, err)
			}

			if got != tc.want {
				t.Errorf("Read(%q) = %q, want %q", tc.src, got, tc.want)
			}
		})
	}
}
