package main

import (
	"bufio"
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
	ClientConfigDir          string
}

func serveWelcome(w http.ResponseWriter, r *http.Request) {
	cn := r.TLS.PeerCertificates[0].Subject.CommonName
	fmt.Fprintln(w, "Welcome", cn, r.URL.Path)
}

// servePublic delivers public files
func servePublic(w http.ResponseWriter, r *http.Request) {

	// do not allow ..
	if strings.Contains(r.URL.Path, "..") {
		http.Error(w, "No `..` allowed.", http.StatusNotAcceptable)
		return
	}

	// construct the filename, hide short-lived variables inside the
	// path: "" / cert /example.com
	// parts 0    1    2
	var fn string
	{
		parts := strings.Split(r.URL.Path, "/")
		fn = filepath.Join(parts[2], parts[1]+".pem")
	}

	file, err := http.Dir(opt.Certbase).Open(fn)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	io.Copy(w, file)
}

func servePrivate(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "..") {
		http.Error(w, "No `..` allowed.", http.StatusNotAcceptable)
		return
	}

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
	ok := func() bool {
		scanner := bufio.NewScanner(config)
		for scanner.Scan() {
			line := strings.Trim(strings.SplitN(scanner.Text(), "#", 2)[0], " \t\r")
			if line == domain {
				return true
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
		return false
	}()

    if !ok {
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
    io.Copy(w, file)

}

func main() {

	http.HandleFunc("/cert/", servePublic)
	http.HandleFunc("/chain/", servePublic)
	http.HandleFunc("/fullchain/", servePublic)
	http.HandleFunc("/privkey/", servePrivate)

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
		ClientAuth:   tls.RequireAndVerifyClientCert, // !! RequireAndVerify !!
		Certificates: []tls.Certificate{cert},
	})
	Check(err)

	http.Serve(listener, nil)
}
