package main

import (
	"crypto/tls"
	"log"
	"net/http"

	"go.schlittermann.de/heiko/cert-proxy/program"
	. "go.schlittermann.de/heiko/cert-proxy/shared"
)

var (
	opt struct { // see init.go for defaults
		Certbase        string
		SSLFile         string
		Serve           string
		ClientConfigDir string
		Verbose         bool
	}
)

type contextKey int

const (
	REMOTE contextKey = iota
	DOMAIN
)

type context map[contextKey]string
type handleFunc func(http.ResponseWriter, *http.Request)
type handleFuncCTX func(context, http.ResponseWriter, *http.Request) error

func use(handlers ...handleFuncCTX) handleFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := make(context)
		for _, f := range handlers {
			if err := f(ctx, w, req); err != nil {
				log.Printf("Remote %v: %v", req.RemoteAddr, err)
				return
			}
		}
	}
}

func main() {

	http.HandleFunc("/v1/list", use(authn, serve))
	http.HandleFunc("/v1/cert/", use(serve))
	http.HandleFunc("/v1/chain/", use(serve))
	http.HandleFunc("/v1/fullchain/", use(serve))
	http.HandleFunc("/v1/privkey/", use(authz, serve))
	http.HandleFunc("/v1/bundle/", use(authz, serve))

	tlsConfig, err := TLSServerConfig(opt.SSLFile, &tls.Config{
		ClientAuth: tls.VerifyClientCertIfGiven,
	})
	if err != nil {
		log.Fatal(err)
	}

	listener, err := tls.Listen("tcp", opt.Serve, tlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting listener %v (%s: %s)\n", listener.Addr(), program.Path, program.Version)
	http.Serve(listener, nil)
}
