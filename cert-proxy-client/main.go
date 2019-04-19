package main

import (
	. "cert-proxy/internal/shared"
	"crypto/tls"
	"log"
	"net/http"
	"path"
	"sync"
)

const API_VERSION = `v1`

var (
	CNs = UList{} // List of unique strings
	opt = struct {
		CNfile    string // the CNs to fetch
		Certbase  string // where to put the output
		Connect   string // Server address
		Jobs      int    // parallel Jobs
		OutFormat Format // PEM|PKCS12
		Outfile   string // ignore certbase and output directly
		SSLFile   string // SSL auth file
		ServerCN  string // X509 CN of the server
		Verbose   bool
		Auto      bool // Fetch all (Issue /list first)
	}{
		Auto:      true,
		OutFormat: FORMAT, // platform dependend, PEM (*nix) vs PKCS12 (Win*)
	}
	verbose func(string, ...interface{})
)

func main() {
	defer verbose("Done")

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
		verbose("Getting list of CNs")
		resp, err := http.Get(opt.Connect + path.Join(`/`+API_VERSION, `list`))
		if err != nil {
			log.Fatal(err)
		}
		AddItemsFromReader(&CNs, resp.Body)
		resp.Body.Close()
	}

	verbose("Enqueing tasks for %d CNs", len(CNs))
	var queue = make(chan Task, opt.Jobs+1)
	go func() {
		enqueTasks(queue, CNs, opt.OutFormat, ITEMS[opt.OutFormat])
		close(queue)
	}()

	verbose("Launching workers")
	var wg = sync.WaitGroup{}
	for i := 0; i < opt.Jobs; i++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			Worker(wid, queue)
		}(i)
	}
	verbose("Waiting for completion")
	wg.Wait()
}
