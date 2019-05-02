package main

import (
	"bufio"
	"cert-proxy/cert-proxy-client/cert"
	"cert-proxy/cert-proxy-client/worker"
	"cert-proxy/internal/list"
	"cert-proxy/internal/program"
	. "cert-proxy/internal/shared"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"path"
)

const API_VERSION = `v1`

var (
	CNs = list.UniqStrings{}
	opt = struct {
		Auto     bool        // Fetch all (Issue /list first)
		Certbase string      // where to put the output
		CNfile   string      // the CNs to fetch
		Connect  string      // Server address
		Format   cert.Format // PEM|PKCS12
		Hook     string      // Hook file
		Jobs     int         // parallel Jobs
		Passout  string      // PKC12 password
		ServerCN string      // X509 CN of the server
		SSLFile  string      // SSL auth file
		Verbose  bool
	}{
		Format: cert.FORMAT, // platform dependend, PEM (*nix) vs PKCS12 (Win*)
	}
)

func main() {
	defer Verbose("DONE")
	Verbose("Starting %s: %s", program.Name, program.Version)

	if err := list.AddItemsFromFile(&CNs, opt.CNfile); err != nil {
		log.Fatal(err)
	}

	// Setup the HTTP client, some more setup is necessary, as we need
	// to send our certificate and we need to check the server's cert
	// agains a non-public root CA.
	// FIXME: is this really the right way?

	http.DefaultClient.Transport = &http.Transport{
		// Go behaves quite rude and just tears down the connections
		// when the program stops (even the CloseIdleConnections
		// doesn't seem to help
		// If we want to be more polite, we should disable the long
		// lived connections:
		//DisableKeepAlives: true,
		TLSClientConfig: func() *tls.Config {
			cfg, err := TLSClientConfig(opt.SSLFile, &tls.Config{
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
		if pushed, err := fetchCNs(); err != nil {
			log.Fatal(err)
		} else {
			CNs.Add(pushed...)
		}
	}

	Verbose("Enqueing tasks for %d domains %v", len(CNs), CNs)

	var pool = worker.NewPool(min(opt.Jobs, len(CNs)))
	pool.EnqueueTasks(CNs, opt.Connect, opt.Certbase, opt.Hook, opt.Format, opt.Passout)
	pool.Wait()

}

func fetchCNs() ([]string, error) {
	Verbose("Getting list of domains")
	resp, err := http.Get(opt.Connect + path.Join(`/`+API_VERSION, `list`))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	defer resp.Body.Close()
	var domains []string

	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		domains = append(domains, s.Text())
	}
	return domains, s.Err()
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}
