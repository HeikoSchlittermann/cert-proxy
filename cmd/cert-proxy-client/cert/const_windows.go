// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package cert

// FORMAT is the platform-default certificate Format (PEM on Unix, PKCS12 on Windows).
const FORMAT Format = FormatPKCS12

// PKCS12Compat is the platform-default PKCS12 compatibility level.
const PKCS12Compat = "legacy"
