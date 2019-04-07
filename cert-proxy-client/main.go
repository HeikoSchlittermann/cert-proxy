package main

import (
	. "cert-proxy/shared"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	CN  string
	opt struct {
		Certbase                 string
		CaFile, CrtFile, KeyFile string
		Connect                  string
		ServerCN                 string
		Outfile                  string
	}
)

func main() {

	urlBase := "https://" + opt.Connect

	// ohoh, even in Go we can write unreadable code, just to
	// avoid having tons of semiglobal variables
	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: func() *tls.Config {
			cert, err := tls.LoadX509KeyPair(opt.CrtFile, opt.KeyFile)
			if err != nil {
				log.Fatal(err)
			}
			return &tls.Config{
				RootCAs:      CertPool(opt.CaFile),
				ServerName:   opt.ServerCN,
				Certificates: []tls.Certificate{cert},
			}
		}(),
	}

	resp, err := http.Get(urlBase + "/cert/" + CN)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Ok, we can get something, so prepare the destination,
	// if any
	var out io.WriteCloser

	switch certFile := opt.Outfile; certFile {
	case "-":
		out = os.Stdout
	case "":
		cnDir := filepath.Join(opt.Certbase, CN)
		err = os.Mkdir(cnDir, 0777)
		if m, e := os.Stat(cnDir); !os.IsExist(err) || e != nil || !m.IsDir() {
			log.Fatal(err)
		}

		certFile = filepath.Join(cnDir, "cert.pem")
		fallthrough
	default:
		out, err = os.Create(certFile)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

}
