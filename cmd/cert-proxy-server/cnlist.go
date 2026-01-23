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
	cc, err := http.Dir(opt.ClientConfigDir).Open(cn)
	if err != nil {
		return nil, err
	}
	defer cc.Close()

	cns := list.UniqStrings{}
	if err = list.AddItemsFromReader(&cns, cc); err != nil {
		return nil, err
	}

	return cns, nil
}
