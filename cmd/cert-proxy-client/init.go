// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"strings"

	"go.schlittermann.de/heiko/cert-proxy/cmd/cert-proxy-client/cert"
	"go.schlittermann.de/heiko/cert-proxy/cmd/cert-proxy-client/secret"
	"go.schlittermann.de/heiko/cert-proxy/internal/program"
	. "go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

func init() {
	// Running as a systemd unit?
	if os.Getenv(`JOURNAL_STREAM`) != "" {
		log.SetFlags(0)
	}

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] [<CN>]...\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprint(flag.CommandLine.Output(), `
The cert-proxy-client exits with 0 on success, and with an non-zero value if there
is a problem.

¹) The hook script gets called *for each* certificate as soon as it is
   done (whether fetched or unmodified):

       <script> deploy_cert <DOMAIN> <KEYFILE> <CERTFILE> <FULLCHAIN> <CHAINFILE> <TIMESTAMP>
    or
       <script> deploy_cert <DOMAIN> <BUNDLEFILE> <TIMESTAMP>

	Additionally the corresponding environment variables are set
	(possibly overriding existing environment variables with the same
	name.

	Note for Windows Powershell: You may wish to "set-executionpolicy remotesigned"

	Note on concurrency: The hooks run sequentially: at no time the
	hook script will run in more than one instance. But, during the hook
	script is running, other threads of the cert-proxy-client may
	replace certificates your hook script relies indirectly on.

²) The shared hook script gets called once *after* all other hooks are done:

      <script> shared <DOMAIN>...

	An environment variable "DOMAINS" is provided too, containing a space separated
	list of the domain names.

³) The password for protecting the P12 file may be given in one of the following notations:
    - pass:<password>
    - file:<file containing the password>
    - env:<environment variable containing the password>

Example:

	cert-proxy-client -connect https://cert-proxy/ \
					  -servernae certs.example.com \
					  -sslfile client-ssl.pem \
					  -verbose
`)
	}

	// On Windows: outFormat is PKCS12 and Item is "bundle"
	// On Linux: outFormat is PEM and Items are "cert", "chain", "fullchain", "privkey"
	// -outformat PEM    implies cert, chain, fullchain, privkey
	// -outformat PKCS12 implies bundle

	var logOutput out = STDERR

	var (
		help    = flag.Bool("help", false, "print help to STDOUT and exit cleanly")
		version = flag.Bool("version", false, "current version ("+program.Version+")")
	)

	flag.BoolVar(&cert.UseSymlink, "symlink", cert.UseSymlink, "use symlinks for current files")
	flag.BoolVar(&opt.Auto, "auto", true, "auto mode (fetch all CNs the server provides us)")
	flag.BoolVar(&opt.Verbose, "verbose", false, "verbose output")
	flag.BoolVar(&cert.Force, "force", false, "force download, even if not modified")
	flag.IntVar(&opt.Jobs, "jobs", runtime.NumCPU(), "maximum number of parallel running `jobs`")
	flag.StringVar(&opt.Certbase, "certbase", "certs", "base `dir` for downloaded certs")
	flag.StringVar(&opt.CNfile, "cnfile", "", "CN list `file` (use - for stdin)")
	flag.StringVar(&opt.Connect, "connect", "https://localhost:4433", "address of cert proxy `[scheme://]server`")
	flag.StringVar(&opt.Hook, "hook", "", "hook script `file`¹")
	flag.StringVar(&opt.Passout, "passout", "", "`password` to protect the PKCS12³")
	flag.StringVar(&opt.SharedHook, "shared-hook", "", "shared hook script `file`²")
	flag.StringVar(&opt.ServerCN, "servername", "", "name (`CN`) of the cert proxy server (if empty: use the FQDN of the host we connect to)")
	flag.StringVar(&opt.SSLFile, "sslfile", "client-ssl.pem", "SSL auth `file` (crt+key+ca) PEM")
	flag.Var(&logOutput, "stderr", "redirect stderr `output` (stderr|stdout)")
	flag.Var(&opt.Format, "format", "`format` of the requested certificate(s) (PEM|PKCS12)")
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

	if logOutput == `STDOUT` {
		*os.Stderr = *os.Stdout
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

	if opt.Passout != "" {
		var err error

		opt.Passout, err = secret.Read(opt.Passout)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Sanitize the Connect option
	if url, err := url.Parse(opt.Connect); err != nil {
		log.Fatal(err)
	} else {
		if url.Scheme == "" {
			url.Scheme = "https"
		}

		url.Path = strings.TrimRight(url.Path, "/")
		opt.Connect = url.String()
	}
}
