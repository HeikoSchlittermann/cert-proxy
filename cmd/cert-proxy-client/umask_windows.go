// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0
// (co)authored by ai:claude-sonnet-4-5

//go:build windows

package main

// hardenUmask is a no-op on Windows, which has no umask concept; file
// access is governed by ACLs instead. On Windows the client defaults to
// PKCS12 output (see cert.FORMAT), so there is no POSIX-mode hardening to
// apply here.
func hardenUmask() {}
