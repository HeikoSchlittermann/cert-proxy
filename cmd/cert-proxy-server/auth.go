// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

// Cert-proxy-server implements the server side of the cert-proxy
// protocol.
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func authn(ctx context, w http.ResponseWriter, req *http.Request) error {
	c := req.TLS.PeerCertificates
	if len(c) == 0 {
		err := errors.New("no (valid) client certificate")
		http.Error(w, err.Error(), http.StatusUnauthorized)

		return err
	}

	ctx[REMOTE] = c[0].Subject.CommonName

	return nil
}
func authz(ctx context, w http.ResponseWriter, req *http.Request) error {
	if err := authn(ctx, w, req); err != nil {
		return err
	}

	// <>/v1/<req>/[domain]
	// 0  1  2     3
	parts := strings.Split(req.URL.Path, `/`)
	if len(parts) < 4 {
		err := errors.New("required syntax: /v1/<req>[/<domain>]")
		http.Error(w, err.Error(), http.StatusBadRequest)

		return err
	}

	ctx[DOMAIN] = parts[3]

	allowedDomains, err := cnList(ctx[REMOTE])
	if err != nil {
		status := http.StatusInternalServerError
		if os.IsNotExist(err) {
			status = http.StatusUnauthorized
		}

		http.Error(w, err.Error(), status)

		return err
	}

	if _, ok := allowedDomains[ctx[DOMAIN]]; !ok {
		err := fmt.Errorf("client %s not authorized for %s", ctx[REMOTE], ctx[DOMAIN])
		http.Error(w, err.Error(), http.StatusUnauthorized)

		return err
	}

	return nil
}
