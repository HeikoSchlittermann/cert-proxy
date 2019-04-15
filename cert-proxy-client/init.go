package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func init() {

	log.SetFlags(log.Flags()|log.Lmicroseconds) // supress Timestamp output

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [<CN>]...\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&opt.SSLFile, "sslfile", "ssl.pem", "SSL auth file (crt+key+ca) PEM")
	flag.StringVar(&opt.Connect, "connect", "https://localhost:4433", "Address of cert proxy server")
	flag.StringVar(&opt.ServerCN, "cert-proxy-cn", "cert-proxy", "CN of the cert proxy certificate")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "Base dir for downloaded certs")
	flag.StringVar(&opt.Outfile, "outfile", "", "Output file (use - for stdout)")
	flag.StringVar(&opt.CNfile, "cnfile", "", "CN list file (use - for stdin)")
	flag.BoolVar(&opt.Verbose, "verbose", false, "Verbose output")
	flag.Var(&opt.OutFormat, "format", "Format of the requested certificate(s)")
	flag.Parse()

	if opt.CNfile == "" && flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	for _, v := range flag.Args() {
		CNs.Add(v)
	}

	if opt.Verbose {
		verbose = func(format string, s ...interface{}) { log.Printf(format, s...) }
	} else {
		verbose = func(string, ...interface{}) {}
	}
}
