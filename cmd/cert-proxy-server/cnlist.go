package main

import (
	"git.schlittermann.de/user/heiko/cert-proxy.git/internal/list"
	"net/http"
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
