package main

import (
	"cert-proxy/internal/program"
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
)

func serveWelcome(w http.ResponseWriter, r *http.Request) {
	cn := r.TLS.PeerCertificates[0].Subject.CommonName
	fmt.Fprintln(w, "Welcome", cn, r.URL.Path)
}

func servePublic(w http.ResponseWriter, r *http.Request) {

	cn := r.TLS.PeerCertificates[0].Subject.CommonName
	parts := strings.Split(r.URL.Path, "/")[2:]
	req, parts := parts[0], parts[1:]
	Verbose("Serving cn=%s %s\n", cn, r.URL)

	// return the list of domains this client is allowed to fetch the
	// certificiates
	if req == "list" {
		if allowedDomains, err := cnList(cn); err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else {
			fmt.Fprintln(w, strings.Join(allowedDomains.Items(), "\n"))
			return
		}
	}

	domain, parts := parts[0], parts[1:]
	fn := filepath.Join(domain, req+".pem")

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
	cn := r.TLS.PeerCertificates[0].Subject.CommonName
	parts := strings.Split(r.URL.Path, "/")[2:]
	Verbose("Serving cn=%s %v", cn, r.URL)

	allowedDomains, err := cnList(cn)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	req, parts := parts[0], parts[1:]
	domain, parts := parts[0], parts[1:]
	filename := filepath.Join(domain, req+func() string {
		switch strings.ToUpper(r.URL.Query().Get("format")) {
		case `PKCS12`:
			return `.p12`
		case `PEM`:
			fallthrough
		default:
			return `.pem`
		}
	}())

	// now check, if the current cn is allowed to access the domain,
	// that is, we check, if the config file (already opened) contains
	// a line with the current domain
	if _, ok := allowedDomains[domain]; !ok {
		log.Printf("Client cn=%s is not authorized for %s\n", cn, domain)
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

	http.HandleFunc("/v1/list", servePublic)
	http.HandleFunc("/v1/cert/", servePublic)
	http.HandleFunc("/v1/chain/", servePublic)
	http.HandleFunc("/v1/fullchain/", servePublic)
	http.HandleFunc("/v1/privkey/", servePrivate)
	http.HandleFunc("/v1/bundle/", servePrivate)

	tlsConfig, err := TLSServerConfig(opt.SSLFile, &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
	})
	if err != nil {
		log.Fatal(err)
	}

	listener, err := tls.Listen("tcp", opt.Serve, tlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting listener %v (%s: %s)\n", listener.Addr(), program.Name, program.Version)
	http.Serve(listener, nil)
}
