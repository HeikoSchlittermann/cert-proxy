package main

import (
	. "cert-proxy/internal/shared"
	"net/http"
)

// cnList reads the client config file and returns
// the list of allowed domains
func cnList(cn string) (UList, error) {

	cc, err := http.Dir(opt.ClientConfigDir).Open(cn)
	if err != nil {
		return nil, err
	}
	defer cc.Close()

	cns := UList{}
	if err = AddItemsFromReader(&cns, cc); err != nil {
		return nil, err
	}

	return cns, nil
}
