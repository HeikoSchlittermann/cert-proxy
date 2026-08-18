// Copyright 2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package shared

import "net/url"

// RedactedURL returns u with the named query parameters replaced, for logging.
// If the raw query is malformed it redacts the complete query: url.URL.Query
// discards parse errors, and returning the original URL in that case could leak
// precisely the value this function is meant to hide.
func RedactedURL(u *url.URL, secretNames ...string) string {
	if u == nil {
		return ""
	}

	if u.RawQuery == "" {
		return u.String()
	}

	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		clone := *u
		clone.RawQuery = "REDACTED"
		clone.ForceQuery = true

		return clone.String()
	}

	var found bool

	for _, name := range secretNames {
		if q.Has(name) {
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
