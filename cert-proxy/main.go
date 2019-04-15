package main

import (
	. "cert-proxy/internal/shared"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

var (
	opt struct { // see init.go for defaults
		Certbase        string
		SSLFile         string
		Serve           string
		ClientConfigDir string
	    Verbose         bool
	}
    verbose func(string, ...interface{})
)

func serveWelcome(w http.ResponseWriter, r *http.Request) {
	cn := r.TLS.PeerCertificates[0].Subject.CommonName
	fmt.Fprintln(w, "Welcome", cn, r.URL.Path)
}

func servePublic(w http.ResponseWriter, r *http.Request) {

	// do not allow .., but as we use http.Dir(), we should be
	// protected, as http.Dir() does not accept .. in the path on it's
	// own. (Otherwise check string.Contains(r.URL.Path, `..`)
	// and return http.StatusNotAcceptable)

	// construct the filename, hide short-lived variables inside the
	// URL.Path: /<type>/<domain>   with type: // (cert|chain|fullchain|privkey|pkcs12)
	// path:  <certbase>/<domain>/<type>.pem
	var fn string = func() string {
		parts := strings.Split(r.URL.Path, "/") // /<type>/<domain>
		return filepath.Join(parts[2], parts[1]+".pem")
	}()

	file, err := http.Dir(opt.Certbase).Open(fn)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer file.Close()

	if _, err := io.Copy(w, file); err != nil {
		log.Fatal(err)
	}
}

func servePrivate(w http.ResponseWriter, r *http.Request) {

	// do not allow .., but as we use http.Dir(), we should be
	// protected, as http.Dir() does not accept .. in the path on it's
	// own. (Otherwise check string.Contains(r.URL.Path, `..`)
	// and return http.StatusNotAcceptable)

	domain, filename := func() (string, string) {
		parts := strings.Split(r.URL.Path, "/")
		return parts[2], filepath.Join(parts[2], parts[1]+".pem")
	}()

	cn := r.TLS.PeerCertificates[0].Subject.CommonName

	// we have a per cn config file in <config-dir>/cn.conf
	config, err := http.Dir(opt.ClientConfigDir).Open(cn)
	if err != nil {
		log.Println(err)
		http.Error(w, "Your cn "+cn+" is unknown", http.StatusForbidden)
		return
	}
	defer config.Close()

	// now check, if the current cn is allowed to access the domain,
	// that is, we check, if the config file (already opened) contains
	// a line with the current domain
	var allowedDomains = UList{}
	err = AddItemsFromReader(&allowedDomains, config)
	if err != nil {
		log.Fatal(err)
	}

	if _, ok := allowedDomains[domain]; !ok {
		log.Printf("%s is not authorized for %s\n", cn, domain)
		http.Error(w, "You are not authorized", http.StatusForbidden)
		return
	}

	file, err := http.Dir(opt.Certbase).Open(filename)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer file.Close()

	if _, err := io.Copy(w, file); err != nil {
		log.Fatal(err)
	}
}

func main() {

	http.HandleFunc("/cert/", servePublic)
	http.HandleFunc("/chain/", servePublic)
	http.HandleFunc("/fullchain/", servePublic)
	http.HandleFunc("/privkey/", servePrivate)

	// The certificate we present to the client
	cert, err := tls.LoadX509KeyPair(opt.SSLFile, opt.SSLFile)
	if err != nil {
		log.Fatal(err)
	}
	pool, err := CertPool(opt.SSLFile)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := tls.Listen("tcp", opt.Serve, &tls.Config{
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert, // !! RequireAndVerify !!
		Certificates: []tls.Certificate{cert},
	})
	if err != nil {
		log.Fatal(err)
	}

    verbose("Starting listener\n")
	http.Serve(listener, nil)
}
