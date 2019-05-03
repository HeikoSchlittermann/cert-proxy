package main

import (
	"bytes"
	"cert-proxy/internal/program"
	. "cert-proxy/internal/shared"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
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

func servePrivate(w http.ResponseWriter, req *http.Request) {

	// do not allow .., but as we use http.Dir(), we should be
	// protected, as http.Dir() does not accept .. in the path on it's
	// own. (Otherwise check string.Contains(req.URL.Path, `..`)
	// and return http.StatusNotAcceptable)
	cn := req.TLS.PeerCertificates[0].Subject.CommonName
	parts := strings.Split(req.URL.Path, "/")[2:]
	Verbose("Serving cn=%s %v", cn, req.URL)

	allowedDomains, err := cnList(cn)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	role, parts := parts[0], parts[1:]
	domain, parts := parts[0], parts[1:]
	filename := filepath.Join(domain, role+func() string {
		switch strings.ToUpper(req.URL.Query().Get("format")) {
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

	var r io.ReadCloser
	if r, err = http.Dir(opt.Certbase).Open(filename); err != nil {
		if os.IsNotExist(err) {
			r, err = createPKCS12(opt.Certbase, domain, req.URL.Query().Get("pass"))
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}
	defer r.Close()

	if _, err := io.Copy(w, r); err != nil {
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

	log.Printf("Starting listener %v (%s: %s)\n", listener.Addr(), program.Path, program.Version)
	http.Serve(listener, nil)
}

func createPKCS12(certbase, domain, pass string) (io.ReadCloser, error) {
	var cert = filepath.Join(certbase, domain, `cert.pem`)
	var key = filepath.Join(certbase, domain, `privkey.pem`)
	var chain = filepath.Join(certbase, domain, `chain.pem`)

	var cmd = exec.Command(`openssl`, `pkcs12`,
		`-export`,
		`-passout`, `pass:` + pass,
		`-inkey`, key,
		`-in`, cert,
		`-certfile`, chain)
	Verbose("Starting %s", cmd.Path)
	pkcs12, err := cmd.Output()
	if err != nil {
		err := err.(*exec.ExitError)
		log.Fatalf("%s %v: %s", cmd.Path, cmd.Args, err.Stderr)
	}
	return ioutil.NopCloser(bytes.NewReader(pkcs12)), err
}
