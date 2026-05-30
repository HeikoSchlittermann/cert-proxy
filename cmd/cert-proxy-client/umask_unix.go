// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0
// (co)authored by ai:claude-sonnet-4-5

//go:build !windows

package main

import (
	"syscall"

	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

// hardenUmask ensures the process umask denies group/other access, so
// certificates and keys written afterwards are not world-readable.
// 0077 = no group/other read/write/execute.
func hardenUmask() {
	currentUmask := syscall.Umask(0)
	syscall.Umask(currentUmask)

	if (currentUmask & 0077) != 0077 {
		shared.Verbose("umask was 0%03o (too permissive); hardening to 0077", currentUmask)
		syscall.Umask(0077)
	}
}
