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

func servePublic(w http.ResponseWriter, r *http.Request) {

	if strings.Contains(r.URL.Path, "..") {
		http.Error(w, "No `..` allowed.", http.StatusNotAcceptable)
		return
	}

    fn := func() string {
        parts := strings.Split(r.URL.Path, "/") // "" / "cert" / "domain"
        return filepath.Join(parts[2], parts[1] + ".pem")
    }()

	file, err := http.Dir(opt.Certbase).Open(fn)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	io.Copy(w, file)
}

func main() {

	http.HandleFunc("/fullchain/", servePublic)
	http.HandleFunc("/chain/", servePublic)
	http.HandleFunc("/cert/", servePublic)
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
