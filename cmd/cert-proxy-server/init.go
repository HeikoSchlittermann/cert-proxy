// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"go.schlittermann.de/heiko/cert-proxy/internal/program"
	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
	"go.schlittermann.de/heiko/cert-proxy/man"
)

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
	flag.DurationVar(&opt.ReadHeaderTimeout, "read-header-timeout", 10*time.Second, "maximum `duration` for reading the request headers (0: no limit)")
	flag.DurationVar(&opt.ReadTimeout, "read-timeout", 30*time.Second, "maximum `duration` for reading the entire request (0: no limit)")
	flag.DurationVar(&opt.WriteTimeout, "write-timeout", 60*time.Second, "maximum `duration` before timing out writes of the response (0: no limit)")
	flag.DurationVar(&opt.IdleTimeout, "idle-timeout", 120*time.Second, "maximum `duration` to wait for the next request on a keep-alive connection (0: use -read-timeout)")
}

// parseTLSVersion maps the -tls-min argument to a crypto/tls version.
func parseTLSVersion(s string) (uint16, error) {
	switch s {
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("-tls-min must be 1.2 or 1.3, got %q", s)
	}
}

func parseFlags() {
	var (
		version = flag.Bool("version", false, "version information ("+program.Version+")")
		help    = flag.Bool("help", false, "print help to STDOUT and exit cleanly")
		tlsMin  = flag.String("tls-min", "1.2", "minimum TLS `version` (1.2|1.3)")
	)

	flag.Parse()

	if *help {
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		os.Exit(0)
	}

	if *version {
		fmt.Println(versionLine())
		os.Exit(0)
	}

	// "man" immediately after the program name is the manual subcommand. After
	// -help and -version, so those keep working with an argument following.
	if args := flag.Args(); man.IsCommand(os.Args) {
		if err := man.Run(man.ServerRegistry(), args[1:]); err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
	}

	var err error

	if opt.TLSMin, err = parseTLSVersion(*tlsMin); err != nil {
		log.Fatal(err)
	}

	// A wrong -certbase would otherwise only show up as 404s for every
	// domain, a wrong -ccd as 401s for every client.
	if err := checkDir(opt.Certbase); err != nil {
		log.Fatalln("-certbase:", err)
	}

	if err := checkDir(opt.ClientConfigDir); err != nil {
		log.Fatalln("-ccd:", err)
	}

	if opt.Verbose {
		shared.EnableVerbose()
	}
}

// checkDir verifies that the directory dir exists and can be read (listed and
// searched) before the listener starts, similar to checkCertbase of
// cert-proxy-client. The caller prefixes the flag name. The server never
// writes, so write access is deliberately not required: under
// ProtectSystem=strict the directories are read-only for the service.
func checkDir(dir string) error {
	fi, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Stat follows symlinks, so a dangling one lands here too; say
			// which of the two it is instead of claiming nothing is there.
			if _, lerr := os.Lstat(dir); lerr == nil {
				return fmt.Errorf("%q is a symlink pointing nowhere", dir)
			}

			return fmt.Errorf("%q does not exist", dir)
		}

		return fmt.Errorf("%q: %w", dir, err)
	}

	if !fi.IsDir() {
		return fmt.Errorf("%q is not a directory", dir)
	}

	// Reading one entry needs read and search permission on the directory.
	// It does not prove that every file below is readable: ACME clients
	// re-create keys 0600 on renewal, which still fails per request.
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("%q is not readable: %w", dir, err)
	}
	defer d.Close() //nolint:errcheck // opened read-only

	if _, err := d.Readdirnames(1); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%q is not readable: %w", dir, err)
	}

	return nil
}
