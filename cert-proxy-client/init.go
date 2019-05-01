package main

import (
	"cert-proxy/cert-proxy-client/cert"
	. "cert-proxy/internal/shared"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

func init() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [<CN>]...\n", os.Args[0])
		flag.PrintDefaults()
	}

	// On Windows: outFormat is PKCS12 and Item is "bundle"
	// On Linux: outFormat is PEM and Items are "cert", "chain", "fullchain", "privkey"
	// -outformat PEM    implies cert, chain, fullchain, privkey
	// -outformat PKCS12 implies bundle

	var version bool

	flag.BoolVar(&opt.Auto, "auto", true, "Auto mode (fetch all CNs the server provides us)")
	flag.BoolVar(&opt.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&version, "version", false, "current version")
	flag.BoolVar(&cert.UseSymlink, "symlink", cert.UseSymlink, "Use symlinks for current files")
	flag.IntVar(&opt.Jobs, "jobs", runtime.NumCPU(), "Maximum number of parallel running jobs")
	flag.StringVar(&opt.CNfile, "cnfile", "", "CN list file (use - for stdin)")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "Base dir for downloaded certs")
	flag.StringVar(&opt.Connect, "connect", "https://localhost:4433", "Address of cert proxy server")
	flag.StringVar(&opt.Hook, "hook", "hook", "hook script")
	flag.StringVar(&opt.SSLFile, "sslfile", "ssl.pem", "SSL auth file (crt+key+ca) PEM")
	flag.StringVar(&opt.ServerCN, "cert-proxy-cn", "cert-proxy", "CN of the cert proxy certificate")
	flag.Var(&opt.Format, "format", "Format of the requested certificate(s) (PEM|PKCS12)")
	flag.Var(&opt.Tick, "tick", "time between re-connects (time Duration)")
	flag.Parse()

	if version {
		fmt.Println(Version)
		os.Exit(0)
	}

	if !opt.Auto && opt.CNfile == "" && flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	for _, v := range flag.Args() {
		CNs.Add(v)
	}

	if opt.Verbose {
		Verbose = log.New(os.Stderr, ``, log.Flags()).Printf
	}
}
