package main

import (
	. "cert-proxy/internal/shared"
	"crypto/tls"
	"log"
	"net/http"
	"sync"
)

var (
	CNs = UList{}
	opt = struct {
		Certbase  string // where to put the output
		CNfile    string // the CNs to fetch
		Connect   string // Server address
		Outfile   string // ignore certbase and output directly
		OutFormat Format // PEM|PKCS12
		ServerCN  string // X509 CN of the server
		SSLFile   string // SSL auth file
		Verbose   bool
	}{OutFormat: FORMAT}
	verbose func(string, ...interface{})
)

func main() {
	defer verbose("done")

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
	defer http.DefaultClient.Transport.(*http.Transport).CloseIdleConnections()

	verbose("Enqueing tasks")
	var queue = make(chan Task)
	go func() {
		enqueTasks(queue, CNs, ITEMS[opt.OutFormat])
		close(queue)
	}()

	verbose("Launching workers")
	var wg = sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			Worker(wid, queue)
		}(i)
	}
	verbose("Waiting for completion")
	wg.Wait()
}
