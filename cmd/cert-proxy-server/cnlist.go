// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net/http"

	"go.schlittermann.de/heiko/cert-proxy/internal/list"
)

// cnList reads the client config file and returns
// the list of allowed domains
func cnList(cn string) (list.UniqStrings, error) {
	// Validate before using as filename — auth.go forwards the
	// returned error to the HTTP client, so it must not echo the CN.
	if err := list.ValidateDomain(cn); err != nil {
		return nil, err
	}

	cc, err := http.Dir(opt.ClientConfigDir).Open(cn)
	if err != nil {
		return nil, err
	}
	defer cc.Close() //nolint:errcheck

	cns := list.UniqStrings{}
	if err = list.AddItemsFromReader(&cns, cc); err != nil {
		return nil, err
	}

	return cns, cc.Close()
}
