package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go.schlittermann.de/heiko/cert-proxy.git/program"
	. "go.schlittermann.de/heiko/cert-proxy.git/shared"
)

func init() {

	// Running as a systemd unit?
	if os.Getenv(`INVOCATION_ID`) != "" {
		log.SetFlags(0)
	}
	//	    log.SetPrefix(filepath.Base(os.Args[0]) + `: `)

	var version = flag.Bool("version", false, "version information ("+program.Version+")")
	var help = flag.Bool("help", false, "print help to STDOUT and exit cleanly")

	flag.StringVar(&opt.SSLFile, "sslfile", "server-ssl.pem", "SSL auth `file` (crt+key+ca) PEM")
	flag.StringVar(&opt.Serve, "serve", ":4433", "listener `[host]:port`")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "base `dir` for certificates")
	flag.StringVar(&opt.ClientConfigDir, "ccd", "clients", "client configuration `dir`")
	flag.BoolVar(&opt.Verbose, "verbose", false, "verbose operation")
	flag.Parse()

	if *help {
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		os.Exit(0)
	}

	if *version {
		fmt.Println(program.Version, program.Name, program.Path)
		os.Exit(0)
	}

	if opt.Verbose {
		Verbose = log.Printf
	} else {
		Verbose = func(string, ...interface{}) {}
	}
}
