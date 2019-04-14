package main

import (
	. "cert-proxy/shared"
	"crypto/tls"
	"log"
	"net/http"
	"sync"
)

var (
	CNs = UList{}
	opt = struct {
		Certbase  string
		SSLFile   string
		Connect   string
		ServerCN  string
		Outfile   string
		CNfile    string
		Verbose   bool
		OutFormat Format
	}{OutFormat: FORMAT}
	verbose func(string, ...interface{})
)

func tlsClientConfig(sslFile string) (config *tls.Config, err error) {
	cert, err := tls.LoadX509KeyPair(sslFile, sslFile)
	if err != nil {
		return
	}
	pool, err := CertPool(opt.SSLFile)
	if err != nil {
		return
	}

	config = &tls.Config{
		RootCAs:      pool,
		ServerName:   opt.ServerCN,
		Certificates: []tls.Certificate{cert},
	}
	return
}

func main() {

	// Build the list of DNs (Domains) we need to fetch the
	// certficates for
	err := AddItemsFromFile(&CNs, opt.CNfile)
	if err != nil {
		log.Fatal(err)
	}

	// Build a list of items to fetch. This depends on the
	// output format (cert, chain, fullchain, privkey), (pkcs12)
	var items = ITEMS[opt.OutFormat]

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
			config, err := tlsClientConfig(opt.SSLFile)
			if err != nil {
				log.Fatal(err)
			}
			return config
		}(),
	}
	// WTF is going on here, I need to explore this in more detail
	defer http.DefaultClient.Transport.(*http.Transport).CloseIdleConnections()

	verbose("Enqueing tasks")
	var queue = make(chan Task)
    go func() {
        enqueTasks(queue, CNs, items)
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
