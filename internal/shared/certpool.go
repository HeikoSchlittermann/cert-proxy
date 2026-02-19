// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

// Package shared provides code used in both the client and the server
package shared

import (
	"crypto/x509"
	"os"
)

// CertPool creates a pool of certificates from PEM files. This is mainly for
// the CA certificate collection
func CertPool(files ...string) (pool *x509.CertPool, err error) {
	pool = x509.NewCertPool()

	for _, f := range files {
		var pem []byte

		pem, err = os.ReadFile(f)
		if err != nil {
			return
		}

		if !pool.AppendCertsFromPEM(pem) {
			panic("Can't append to ca cert pool")
		}
	}

	return
}
