// Copyright 2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactedURL(t *testing.T) {
	tests := []struct {
		name, in string
		want     string
	}{
		{"no query", "/v1/cert/example.com", "/v1/cert/example.com"},
		{"harmless query", "/v1/bundle/example.com?format=PKCS12", "/v1/bundle/example.com?format=PKCS12"},
		{"password", "/v1/bundle/example.com?pass=hunter2", "/v1/bundle/example.com?pass=REDACTED"},
		{
			"password among others",
			"/v1/bundle/example.com?format=PKCS12&pass=hunter2&pkcs12-compat=legacy",
			"/v1/bundle/example.com?format=PKCS12&pass=REDACTED&pkcs12-compat=legacy",
		},
		{"empty password still redacted", "/v1/bundle/example.com?pass=", "/v1/bundle/example.com?pass=REDACTED"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.in)
			require.NoError(t, err)

			got := redactedURL(u)
			assert.Equal(t, tc.want, got)
			assert.NotContains(t, got, "hunter2")
		})
	}

	assert.Empty(t, redactedURL(nil))
}
