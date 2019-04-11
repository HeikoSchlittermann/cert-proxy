package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func init() {

	log.SetFlags(0) // supress Timestamp output

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <CN>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&opt.SSLFile, "auth", "proxy.pem", "proxy auth file (crt+key+ca) PEM")
	flag.StringVar(&opt.Connect, "connect", "localhost:4433", "address of cert proxy server")
	flag.StringVar(&opt.ServerCN, "server name", "cert-proxy", "CN of the expected proxy server certificate")
	flag.StringVar(&opt.Certbase, "certbase", CERTBASE, "base dir for downloaded certs")
	flag.StringVar(&opt.Outfile, "outfile", "", "output file (use - for stdout)")
	flag.BoolVar(&opt.Verbose, "verbose", false, "verbose output")
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	CN = flag.Arg(0)

	if opt.Verbose {
		verbose = func(format string, s ...interface{}) { log.Printf(format, s...) }
	} else {
		verbose = func(string, ...interface{}) {}
	}
}
