package main

import (
    . "cert-proxy/internal/shared"
	"flag"
	"log"
)

func init() {
	log.SetFlags(log.Flags() | log.Lmicroseconds)
	flag.StringVar(&opt.SSLFile, "sslfile", "ssl.pem", "SSL auth file (crt+key+ca) PEM")
	flag.StringVar(&opt.Serve, "serve", ":4433", "Listener [host]:port")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "Base dir for certificates")
	flag.StringVar(&opt.ClientConfigDir, "ccd", "clients", "Client configuration dir")
	flag.BoolVar(&opt.Verbose, "verbose", false, "Verbose operation")
	flag.Parse()

	if opt.Verbose {
		Verbose = log.Printf
	} else {
		Verbose = func(string, ...interface{}) {}
	}
}
