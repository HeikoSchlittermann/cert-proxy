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
	"time"
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
		Tick     duration
		Jobs     int    // parallel Jobs
		ServerCN string // X509 CN of the server
		SSLFile  string // SSL auth file
		Verbose  bool
	}{
		Format: cert.FORMAT, // platform dependend, PEM (*nix) vs PKCS12 (Win*)
		Tick:   duration(24 * time.Hour),
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

	// Start the ticker now, to be independend on the runtime of
	// the jobs
	var ticker <-chan time.Time
	for {
		now := time.Now()

		if ticker != nil {
			Verbose("Next run %s", now.Add(time.Duration(opt.Tick)).Format(time.RFC1123))
			<-ticker
		} else if opt.Tick > 0 {
			ticker = time.Tick(time.Duration(opt.Tick))
		}

		// Build the list of DNs (Domains) we need to fetch the
		// certficates for

		// In auto mode fetch the list of available domains and
		// append this list to the static list we may have in CNs
		var domains list.UniqStrings
		if opt.Auto {
			domains = CNs.Copy()
			if pushed, err := fetchCNs(); err != nil {
				log.Print(err)
				continue
			} else {
				domains.Add(pushed...)
			}
		} else {
			domains = CNs
		}

		Verbose("Enqueing tasks for %d domains %v", len(domains), domains)

		var pool = worker.NewPool(min(opt.Jobs, len(domains)))
		pool.EnqueueTasks(domains, opt.Connect, opt.Certbase, opt.Hook, opt.Format)
		pool.Wait()

		if ticker == nil {
			break
		}
	}

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
