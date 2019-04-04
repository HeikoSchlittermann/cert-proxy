package main

import (
	. "cert-proxy/shared"
	"crypto/tls"
	"fmt"
	"net/http"
)

var (
	caFile, crtFile, keyFile string
	serve                    string
)

func serveWelcome(w http.ResponseWriter, r *http.Request) {
	state := r.TLS
	fmt.Fprintf(w, "Welcome %s\r\n",
        state.PeerCertificates[0].Subject.CommonName)
}

func main() {

	http.HandleFunc("/", serveWelcome)

	// The certificate we present to the client
	cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
	Check(err)

	listener, err := tls.Listen("tcp", serve, &tls.Config{
		ClientCAs:    CertPool(caFile),
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
	})
	Check(err)

	http.Serve(listener, nil)
}
