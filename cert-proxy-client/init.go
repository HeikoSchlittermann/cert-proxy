package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

/*
type Items []string

func (items Items) String() string {
	return strings.Join(items, ", ")
}
func (items *Items) Set(value string) error {
    *items = append(*items, value)
    return nil
}

var items = []string{"cert", "chain", "fullchain", "privkey"}
*/

func init() {

	log.SetFlags(0) // supress Timestamp output

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <CN>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&opt.CrtFile, "crt", "crt.pem", "client certificate file")
	flag.StringVar(&opt.KeyFile, "key", "key.pem", "client certificate key file")
	flag.StringVar(&opt.CAFile, "ca", "ca.pem", "client certificate key file")
	flag.StringVar(&opt.Connect, "connect", "localhost:4433", "address of cert proxy server")
	flag.StringVar(&opt.ServerCN, "server name", "cert-proxy", "CN of the server")
	flag.StringVar(&opt.Certbase, "certbase", "/var/lib/dehydrated/certs", "base dir for certs")
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
