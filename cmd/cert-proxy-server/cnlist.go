// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"net/http"
	"strings"

	"go.schlittermann.de/heiko/cert-proxy/internal/list"
)

// errInvalidCN is returned for a CN that cannot safely be used as
// a filename in the clients config dir. The message deliberately
// omits the CN, since auth.go forwards the error to the HTTP client.
var errInvalidCN = errors.New("invalid client CN")

// cnList reads the client config file and returns
// the list of allowed domains
func cnList(cn string) (list.UniqStrings, error) {
	if cn == "" || cn[0] == '.' || strings.ContainsAny(cn, `/\`+"\x00") {
		return nil, errInvalidCN
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
