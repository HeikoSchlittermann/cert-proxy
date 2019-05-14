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

	var printVersion bool
	flag.StringVar(&opt.SSLFile, "sslfile", "server-ssl.pem", "SSL auth file (crt+key+ca) PEM")
	flag.StringVar(&opt.Serve, "serve", ":4433", "Listener [host]:port")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "Base dir for certificates")
	flag.StringVar(&opt.ClientConfigDir, "ccd", "clients", "Client configuration dir")
	flag.BoolVar(&opt.Verbose, "verbose", false, "Verbose operation")
	flag.BoolVar(&printVersion, "version", false, "Version information ("+program.Version+")")
	flag.Parse()

	if printVersion {
		fmt.Println(program.Version, program.Name, program.Path)
		os.Exit(0)
	}

	if opt.Verbose {
		Verbose = log.Printf
	} else {
		Verbose = func(string, ...interface{}) {}
	}
}
