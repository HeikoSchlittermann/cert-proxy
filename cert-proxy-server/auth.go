package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func authn(ctx context, w http.ResponseWriter, req *http.Request) error {
	if c := req.TLS.PeerCertificates; len(c) == 0 {
		err := errors.New("no (valid) cert certificate")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	} else {
		ctx[REMOTE] = c[0].Subject.CommonName
	}

	return nil
}
func authz(ctx context, w http.ResponseWriter, req *http.Request) error {
	if err := authn(ctx, w, req); err != nil {
		return err
	}

	// <>/v1/<req>/[domain]
	// 0  1  2     3
	if parts := strings.Split(req.URL.Path, `/`); len(parts) < 4 {
		err := errors.New("required syntax: /v1/<req>[/<domain>]")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	} else {
		ctx[DOMAIN] = parts[3]
	}

	allowedDomains, err := cnList(ctx[REMOTE])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	if _, ok := allowedDomains[ctx[DOMAIN]]; !ok {
		err := fmt.Errorf("client %s not authorized for %s", ctx[REMOTE], ctx[DOMAIN])
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	return nil
}
