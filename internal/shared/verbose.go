// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package shared //nolint:revive // I know, it is stupid

// Verbose holds the function for verbose output (progress, logging, …)
var Verbose = func(string, ...interface{}) {}
