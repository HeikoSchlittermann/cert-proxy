// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go.schlittermann.de/heiko/cert-proxy/internal/program"
	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
	"go.schlittermann.de/heiko/cert-proxy/man"
)

// manCommand is the positional argument selecting the manual subcommand.
const manCommand = "man"

func init() {
	// Running as a systemd unit?
	if os.Getenv(`INVOCATION_ID`) != "" {
		log.SetFlags(0)
	}

	flag.StringVar(&opt.SSLFile, "sslfile", "server-ssl.pem", "SSL auth `file` (crt+key+ca) PEM")
	flag.StringVar(&opt.Serve, "serve", ":4433", "listener `[host]:port`")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "base `dir` for certificates")
	flag.StringVar(&opt.ClientConfigDir, "ccd", "clients", "client configuration `dir`")
	flag.BoolVar(&opt.Verbose, "verbose", false, "verbose operation")
}

func parseFlags() {
	var (
		version = flag.Bool("version", false, "version information ("+program.Version+")")
		help    = flag.Bool("help", false, "print help to STDOUT and exit cleanly")
	)

	flag.Parse()

	// "man" as the first positional argument is the manual subcommand.
	if args := flag.Args(); len(args) > 0 && args[0] == manCommand {
		if err := man.Run(man.ServerRegistry(), args[1:]); err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
	}

	if *help {
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		os.Exit(0)
	}

	if *version {
		fmt.Println(versionLine())
		os.Exit(0)
	}

	if opt.Verbose {
		shared.EnableVerbose()
	}
}
