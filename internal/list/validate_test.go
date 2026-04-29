// Copyright 2019-2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDomain(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.example.com",
		"*.example.com",
		"a-b_c.example.com",
		"xn--nxasmq6b.example.com",
		"123.456.example.com",
		"a.b.c.d.e.example.com",
	}

	for _, d := range valid {
		t.Run("valid/"+d, func(t *testing.T) {
			require.NoError(t, ValidateDomain(d))
		})
	}

	invalid := []struct {
		name   string
		domain string
	}{
		{"empty", ""},
		{"path_traversal", "../../etc/passwd"},
		{"slash", "foo/bar"},
		{"backslash", "foo\\bar"},
		{"shell_subst", "$(rm -rf /)"},
		{"backtick", "`id`"},
		{"space", "foo bar"},
		{"null_byte", "foo\x00bar"},
		{"semicolon", "foo;bar"},
		{"pipe", "foo|bar"},
		{"ampersand", "foo&bar"},
		{"leading_dot", ".hidden"},
		{"trailing_dot", "example.com."},
		{"double_dot", "foo..bar"},
		{"newline", "foo\nbar"},
		{"tab", "foo\tbar"},
		{"dollar", "foo$bar"},
		{"curly_brace", "foo{bar}"},
		{"angle_bracket", "<script>"},
	}

	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			err := ValidateDomain(tc.domain)
			assert.Error(t, err, "domain %q should be rejected", tc.domain)
		})
	}
}
