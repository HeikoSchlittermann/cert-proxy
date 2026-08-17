// Copyright 2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/url"

	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

func redactedURL(u *url.URL) string {
	return shared.RedactedURL(u, "pass")
}
