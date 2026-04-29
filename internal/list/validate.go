// Copyright 2019-2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"fmt"
	"regexp"
	"strings"
)

var validDomain = regexp.MustCompile(`^[a-zA-Z0-9*._-]+$`)

// ValidateDomain checks that domain is a plausible DNS name safe
// for use in file paths and shell arguments. It rejects path traversal
// sequences, shell metacharacters, and empty strings.
func ValidateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("empty domain name")
	}

	if !validDomain.MatchString(domain) {
		return fmt.Errorf("contains invalid characters")
	}

	for _, label := range strings.Split(domain, ".") {
		if label == "" {
			return fmt.Errorf("contains empty label (double dot or leading/trailing dot)")
		}
	}

	return nil
}
