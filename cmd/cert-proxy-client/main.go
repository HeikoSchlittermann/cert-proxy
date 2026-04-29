// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"go.schlittermann.de/heiko/cert-proxy/cmd/cert-proxy-client/cert"
	"go.schlittermann.de/heiko/cert-proxy/cmd/cert-proxy-client/worker"
	"go.schlittermann.de/heiko/cert-proxy/internal/list"
	"go.schlittermann.de/heiko/cert-proxy/internal/program"
	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

const apiVersion = `v1`

var (
	CNs = list.UniqStrings{}
	opt = struct {
		Auto       bool        // Fetch all (Issue /list first)
		Certbase   string      // where to put the output
		CNfile     string      // the CNs to fetch
		Connect    string      // Server address
		Format     cert.Format // PEM|PKCS12
		Hook       string      // Hook file
		Jobs       int         // parallel Jobs
		Passout    string      // PKC12 password
		SharedHook string      // Shared hook file
		ServerCN   string      // X509 CN of the server
		SSLFile    string      // SSL auth file
		Verbose    bool
	}{
		Format: cert.FORMAT, // platform dependend, PEM (*nix) vs PKCS12 (Win*)
	}
)

func main() {
	parseFlags()
	defer shared.Verbose("DONE")

	shared.Verbose("Starting %s: %s", program.Name, program.Version)

	if err := list.AddItemsFromFile(&CNs, opt.CNfile); err != nil {
		log.Fatal(err)
	}

	// Setup the HTTP client, some more setup is necessary, as we need
	// to send our certificate and we need to check the server's cert
	// against a non-public root CA.
	// FIXME: is this really the right way?

	http.DefaultClient.Transport = &http.Transport{
		// Go behaves quite rude and just tears down the connections
		// when the program stops (even the CloseIdleConnections
		// doesn't seem to help
		// If we want to be more polite, we should disable the long
		// lived connections:
		//DisableKeepAlives: true,
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: func() *tls.Config {
			cfg, err := shared.TLSClientConfig(opt.SSLFile, &tls.Config{
				ServerName: opt.ServerCN,
			})
			if err != nil {
				log.Fatal(err)
			}

			return cfg
		}(),
	}
	// WTF is going on here, I need to explore this in more detail
	// And this CloseIdleConnections doesn't seem to help either
	defer http.DefaultClient.Transport.(*http.Transport).CloseIdleConnections()

	// Build the list of DNs (Domains) we need to fetch the
	// certficates for

	// In auto mode fetch the list of available domains and
	// append this list to the static list we may have in CNs
	if opt.Auto {
		pushed, err := fetchCNs()
		if err != nil {
			log.Fatal(err)
		}

		CNs.Add(pushed...)
	}

	// Now we get all startup information and can start working
	// in parallel
	opt.Jobs = min(opt.Jobs, len(CNs))
	shared.Verbose("Enqueing %d tasks for %d domains %v", opt.Jobs, len(CNs), CNs)

	var pool = worker.NewPool(opt.Jobs)
	pool.EnqueueTasks(CNs, opt.Connect, opt.Certbase, opt.Hook, opt.Format, opt.Passout)

	if err := pool.Wait(); err != nil {
		log.Fatal(err)
	}

	if opt.SharedHook != "" {
		shared.Verbose("Shared hook %s for %s", opt.SharedHook, CNs)

		cmd := exec.Cmd{
			Path:   opt.SharedHook,
			Args:   append([]string{opt.SharedHook, "shared"}, CNs.Items()...),
			Env:    append(os.Environ(), "DOMAINS="+strings.Join(CNs.Items(), " ")),
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		if err := cmd.Run(); err != nil {
			log.Fatalf("Running shared hook %q: %v", opt.SharedHook, err)
		}
	}

	os.Exit(0)
}

func fetchCNs() ([]string, error) {
	shared.Verbose("Getting list of domains")

	req, err := http.NewRequest(`GET`, opt.Connect+path.Join(`/`+apiVersion, `list`), nil)
	req.Header.Add(`x-version`, program.Version)

	if err != nil {
		return nil, err
	}
	//resp, err := http.Get(opt.Connect + path.Join(`/`+API_VERSION, `list`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	defer resp.Body.Close() //nolint:errcheck

	var domains []string

	// check remote version
	if program.Version != resp.Header.Get(`x-version`) {
		log.Printf("Warning: Version mismatch: server:%s client:%s",
			resp.Header.Get(`x-version`), program.Version)
	}

	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		domains = append(domains, s.Text())
	}

	return domains, s.Err()
}
