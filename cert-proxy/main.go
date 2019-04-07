package main

import (
	. "cert-proxy/shared"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

var opt struct { // see init.go for defaults
	Certbase                 string
	CAFile, CrtFile, KeyFile string
	Serve                    string
}

func serveWelcome(w http.ResponseWriter, r *http.Request) {
	cn := r.TLS.PeerCertificates[0].Subject.CommonName
	fmt.Fprintln(w, "HALLO: ", cn, r.URL.Path)
}

func serveCrt(w http.ResponseWriter, r *http.Request) {

	if strings.Contains(r.URL.Path, "..") {
		http.Error(w, "No `..` allowed.", http.StatusNotAcceptable)
		return
	}

	dir := http.Dir(opt.Certbase)
	fn := filepath.Join(strings.TrimPrefix(r.URL.Path, "/cert/"), "cert.pem")
	file, err := dir.Open(fn)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	io.Copy(w, file)
}

func main() {

	http.HandleFunc("/cert/", serveCrt)
	http.HandleFunc("/", serveWelcome)

	// The certificate we present to the client
	cert, err := tls.LoadX509KeyPair(opt.CrtFile, opt.KeyFile)
	if err != nil {
        log.Fatal(err)
    }
    pool, err := CertPool(opt.CAFile)
    if err != nil {
        log.Fatal(err)
    }

	listener, err := tls.Listen("tcp", opt.Serve, &tls.Config{
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
	})
	Check(err)

	http.Serve(listener, nil)
}
