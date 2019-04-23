package main

import (
	"cert-proxy/cert-proxy-client/cert"
	"cert-proxy/cert-proxy-client/worker"
	. "cert-proxy/internal/shared"
	"crypto/tls"
	"log"
	"net/http"
	"path"
)

const API_VERSION = `v1`

var (
	CNs = UList{} // List of unique strings
	opt = struct {
		CNfile   string      // the CNs to fetch
		Certbase string      // where to put the output
		Connect  string      // Server address
		Jobs     int         // parallel Jobs
		Format   cert.Format // PEM|PKCS12
		Outfile  string      // ignore certbase and output directly
		SSLFile  string      // SSL auth file
		ServerCN string      // X509 CN of the server
		Verbose  bool
		Hook     string // Hook file
		Auto     bool   // Fetch all (Issue /list first)
	}{
		Auto:   true,
		Format: cert.FORMAT, // platform dependend, PEM (*nix) vs PKCS12 (Win*)
	}
)

func main() {
	defer Verbose("DONE")

	// Build the list of DNs (Domains) we need to fetch the
	// certficates for
	if err := AddItemsFromFile(&CNs, opt.CNfile); err != nil {
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

	// In auto mode we fetch the list of domains from the proxy and add
	// it to our own (possibly empty) list
	if opt.Auto {
		Verbose("Getting list of CNs")
		resp, err := http.Get(opt.Connect + path.Join(`/`+API_VERSION, `list`))
		if err != nil {
			log.Fatal(err)
		}
		AddItemsFromReader(&CNs, resp.Body)
		resp.Body.Close()
	}

	Verbose("Enqueing tasks for %d CNs", len(CNs))

	var pool = worker.NewPool(opt.Jobs)
	pool.EnqueueTasks(CNs, opt.Connect, opt.Certbase, opt.Format)
	pool.Wait()

}
