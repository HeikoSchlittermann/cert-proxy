// Copyright 2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package man

import "os"

// stdoutIsTerminal reports whether standard output is a character device.
// Cert-proxy has no terminal dependency, so this stays a Stat() check
// rather than pulling in golang.org/x/term for one bit of information.
func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return fi.Mode()&os.ModeCharDevice != 0
}
