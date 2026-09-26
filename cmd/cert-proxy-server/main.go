// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	gocontext "context"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.schlittermann.de/heiko/cert-proxy/internal/program"
	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

var (
	opt struct { // see init.go for defaults
		Certbase        string
		SSLFile         string
		Serve           string
		ClientConfigDir string
		Verbose         bool
		TLSMin          uint16
		// HTTP server timeouts, see net/http.Server
		ReadHeaderTimeout time.Duration
		ReadTimeout       time.Duration
		WriteTimeout      time.Duration
		IdleTimeout       time.Duration
	}
)

// shutdownTimeout bounds the graceful shutdown on SIGTERM/SIGINT.
const shutdownTimeout = 10 * time.Second

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

// newMux returns the request router of the v1 API.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/list", use(authn, serve))
	mux.HandleFunc("/v1/cert/", use(serve))
	mux.HandleFunc("/v1/chain/", use(serve))
	mux.HandleFunc("/v1/fullchain/", use(serve))
	mux.HandleFunc("/v1/privkey/", use(authz, serve))
	mux.HandleFunc("/v1/bundle/", use(authz, serve))

	return mux
}

func main() {
	parseFlags()

	tlsConfig, err := shared.TLSServerConfig(opt.SSLFile, &tls.Config{
		ClientAuth: tls.VerifyClientCertIfGiven,
		MinVersion: opt.TLSMin,
	})
	if err != nil {
		log.Fatal(err)
	}

	listener, err := tls.Listen("tcp", opt.Serve, tlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Handler:           newMux(),
		ReadHeaderTimeout: opt.ReadHeaderTimeout,
		ReadTimeout:       opt.ReadTimeout,
		WriteTimeout:      opt.WriteTimeout,
		IdleTimeout:       opt.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(gocontext.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	served := make(chan error, 1)

	log.Printf("Starting listener %v (%s: %s)\n", listener.Addr(), program.Path, program.Version)

	go func() { served <- srv.Serve(listener) }()

	select {
	case err := <-served:
		log.Fatal(err)
	case <-ctx.Done():
	}

	stop()
	log.Print("shutting down")

	shutdownCtx, cancel := gocontext.WithTimeout(gocontext.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}

	if err := <-served; err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("serve: %v", err)
	}
}
