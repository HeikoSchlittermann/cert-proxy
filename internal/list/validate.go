// Copyright 2019-2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"fmt"
	"regexp"
	"strings"
)

var validDomain = regexp.MustCompile(`^[a-zA-Z0-9*._-]+$`)

// windowsReserved lists DOS/Windows reserved device names. On Windows,
// these resolve to devices regardless of extension or case, so a file
// named CON.txt opens the console. Keeping the set here means client
// names safe on Linux are also safe under a Windows-built binary.
var windowsReserved = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM0": {}, "COM1": {}, "COM2": {}, "COM3": {}, "COM4": {},
	"COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT0": {}, "LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {},
	"LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

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

// ValidateClientName checks that name is safe to use as a filename
// in the per-client config directory on any supported OS, including
// Windows. It enforces every rule of ValidateDomain plus rejection of
// Windows reserved device names and the wildcard character.
func ValidateClientName(name string) error {
	if err := ValidateDomain(name); err != nil {
		return err
	}

	if strings.ContainsRune(name, '*') {
		return fmt.Errorf("contains invalid characters")
	}

	base := name
	if i := strings.IndexByte(name, '.'); i >= 0 {
		base = name[:i]
	}

	if _, reserved := windowsReserved[strings.ToUpper(base)]; reserved {
		return fmt.Errorf("reserved name")
	}

	return nil
}
