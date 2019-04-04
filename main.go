package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	crtFile string
	keyFile string
	caFile  string
	serve   string
)

func check(err error) {
	if err == nil {
		return
	}
	log.Fatal(err)
}

func welcome(w http.ResponseWriter, r *http.Request) {
	state := r.TLS
	fmt.Println(state.PeerCertificates)
	fmt.Println(state.VerifiedChains)
	fmt.Fprintf(w, "Welcome\r\n")
}

func rootCAs(files ...string) (pool *x509.CertPool) {
	pool = x509.NewCertPool()
	for _, f := range files {
		pem, err := ioutil.ReadFile(f)
		check(err)
		if !pool.AppendCertsFromPEM(pem) {
			panic("Can't append to ca cert pool")
		}
	}
	return
}

func main() {

	http.HandleFunc("/", welcome)

	cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
	check(err)

	listener, err := tls.Listen("tcp", serve, &tls.Config{
		ClientCAs:    rootCAs(caFile),
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
	})
	check(err)

	http.Serve(listener, nil)
	/*
		err := http.ListenAndServeTLS(serve, crtFile, keyFile, &tls.Config{})
		check(err)
	*/
}
