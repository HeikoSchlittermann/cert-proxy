package main

import (
	"flag"
)

func init() {
	flag.StringVar(&opt.SSLFile, "sslfile", "ssl.pem", "SSL auth file (crt+key+ca) PEM")
	flag.StringVar(&opt.Serve, "serve", ":4433", "Listener [host]:port")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "Base dir for certificates")
	flag.StringVar(&opt.ClientConfigDir, "ccd", "clients", "Client configuration dir")
	flag.Parse()
}
