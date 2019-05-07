package main

import (
	"bytes"
	"cert-proxy/internal/program"
	. "cert-proxy/internal/shared"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

func servePublic(w http.ResponseWriter, req *http.Request) {

	cn := req.TLS.PeerCertificates[0].Subject.CommonName
	parts := strings.Split(req.URL.Path, "/")[2:]
	role, parts := parts[0], parts[1:]
	Verbose("Serving cn=%s %s ims:%s\n", cn, req.URL, req.Header.Get(`if-modified-since`))

	versionCheck(w, req)

	// return the list of domains this client is allowed to fetch the
	// certificiates
	if role == "list" {
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
	fn := filepath.Join(domain, role+".pem")

	var content io.ReadSeeker
	var mtime time.Time

	if file, err := http.Dir(opt.Certbase).Open(fn); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else {
		defer file.Close()
		if fi, err := file.Stat(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			mtime = fi.ModTime()
		}
		content = file
	}

	http.ServeContent(w, req, domain, mtime, content)
}

func servePrivate(w http.ResponseWriter, req *http.Request) {

	// do not allow .., but as we use http.Dir(), we should be
	// protected, as http.Dir() does not accept .. in the path on it's
	// own. (Otherwise check string.Contains(req.URL.Path, `..`)
	// and return http.StatusNotAcceptable)
	cn := req.TLS.PeerCertificates[0].Subject.CommonName
	parts := strings.Split(req.URL.Path, "/")[2:]
	Verbose("Serving cn=%s %s ims:%s\n", cn, req.URL, req.Header.Get(`if-modified-since`))

	versionCheck(w, req)
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

	var content io.ReadSeeker
	var mtime time.Time

	if file, err := http.Dir(opt.Certbase).Open(filename); err != nil {
		if os.IsNotExist(err) {
			content, mtime, err = createPKCS12(opt.Certbase, domain, req.URL.Query().Get("pass"))
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	} else {
		defer file.Close()
		if fi, err := file.Stat(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			mtime = fi.ModTime()
		}
		content = file
	}

	http.ServeContent(w, req, domain, mtime, content)

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

func createPKCS12(certbase, domain, pass string) (*bytes.Reader, time.Time, error) {

	var cert = filepath.Join(certbase, domain, `cert.pem`)
	var key = filepath.Join(certbase, domain, `privkey.pem`)
	var chain = filepath.Join(certbase, domain, `chain.pem`)
	var mtime time.Time

	// Get the symlinked names
	adjustPath(&cert, &key, &chain)

	if fi, err := os.Stat(cert); err != nil {
		return nil, mtime, err
	} else {
		mtime = fi.ModTime()
	}

	var cmd = exec.Command(`openssl`, `pkcs12`,
		`-export`,
		`-passout`, `pass:`+pass,
		`-inkey`, key,
		`-in`, cert,
		`-certfile`, chain)
	Verbose("Starting %s", cmd.Path)
	pkcs12, err := cmd.Output()
	if err != nil {
		err := err.(*exec.ExitError)
		log.Printf("%s %v: %s", cmd.Path, cmd.Args, err.Stderr)
	}
	return bytes.NewReader(pkcs12), mtime, err
}

// If the first item has an infix (-<xxxxx>.pem), we adjust all names to
// posess this infix

func adjustPath(names ...*string) {
	l, err := os.Readlink(*names[0])

	if err != nil {
		return
	} // not a link

	infix := l[strings.LastIndex(l, `-`):strings.LastIndex(l, `.`)]

	for _, v := range names {
		dot := strings.LastIndex(*v, `.`)
		*v = (*v)[0:dot] + infix + (*v)[dot:]
	}

}

func versionCheck(w http.ResponseWriter, req *http.Request) {
	if program.Version != req.Header.Get(`x-version`) {
		log.Printf("Version mismatch: server:%s client:%s",
			program.Version, req.Header.Get(`x-version`))
	}
	w.Header().Add(`x-version`, program.Version)
}
