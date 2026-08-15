// Copyright 2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import "net/url"

// redactSecret lists query parameters that must never be logged.
var redactSecret = map[string]bool{"pass": true}

// redactedURL returns u with secret query parameters replaced, for logging.
// A PKCS12 bundle may be requested with ?pass=..., and verbose operation
// would otherwise copy that password into the journal.
func redactedURL(u *url.URL) string {
	if u == nil {
		return ""
	}

	q := u.Query()

	var found bool

	for name := range q {
		if redactSecret[name] {
			q.Set(name, "REDACTED")

			found = true
		}
	}

	if !found {
		return u.String()
	}

	clone := *u
	clone.RawQuery = q.Encode()

	return clone.String()
}
