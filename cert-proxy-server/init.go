package main

import (
	"cert-proxy/internal/program"
	. "cert-proxy/internal/shared"
	"flag"
	"fmt"
	"log"
	"os"
)

func init() {

	// Running as a systemd unit?
	if os.Getenv(`INVOCATION_ID`) != "" {
		log.SetFlags(0)
	}
	//	    log.SetPrefix(filepath.Base(os.Args[0]) + `: `)

	var version bool
	flag.BoolVar(&version, "version", false, "current version")
	flag.StringVar(&opt.SSLFile, "sslfile", "ssl.pem", "SSL auth file (crt+key+ca) PEM")
	flag.StringVar(&opt.Serve, "serve", ":4433", "Listener [host]:port")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "Base dir for certificates")
	flag.StringVar(&opt.ClientConfigDir, "ccd", "clients", "Client configuration dir")
	flag.BoolVar(&opt.Verbose, "verbose", false, "Verbose operation")
	flag.Parse()

	if version {
		fmt.Println(program.Version)
		os.Exit(0)
	}

	if opt.Verbose {
		Verbose = log.Printf
	} else {
		Verbose = func(string, ...interface{}) {}
	}
}
