package main

import (
	"flag"
)

func init() {
	flag.StringVar(&opt.CrtFile, "crt", "crt.pem", "certificate file")
	flag.StringVar(&opt.KeyFile, "key", "key.pem", "key file")
	flag.StringVar(&opt.CAFile, "ca", "ca.pem", "CA file")
	flag.StringVar(&opt.Serve, "serve", ":4433", "[host]:port for listener")
	flag.StringVar(&opt.Certbase, "certbase", "/var/lib/dehydrated/certs", "cert base dir")
	flag.StringVar(&opt.ClientConfigDir, "ccd", "clients", "client configuration dir")
	flag.Parse()
}
