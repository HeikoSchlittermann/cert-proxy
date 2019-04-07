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
		CAFile, CrtFile, KeyFile string
		Connect                  string
		ServerCN                 string
		Outfile                  string
		Verbose                  bool
	}
	verbose func(string, ...interface{})
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
            pool, err := CertPool(opt.CAFile)
            if err != nil {
                log.Fatal(err)
            }
			return &tls.Config{
				RootCAs:      pool,
				ServerName:   opt.ServerCN,
				Certificates: []tls.Certificate{cert},
			}
		}(),
	}

	URL := urlBase + "/cert/" + CN
	verbose("Getting %s", URL)

	resp, err := http.Get(URL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Ok, we can get something, so prepare the destination,
	var out io.WriteCloser
	switch certfile := opt.Outfile; certfile {
	case "-":
		out = os.Stdout
		verbose("output to STDOUT")
	case "":
		cnDir := filepath.Join(opt.Certbase, CN)
		err = os.Mkdir(cnDir, 0777)
		if m, e := os.Stat(cnDir); !os.IsExist(err) || e != nil || !m.IsDir() {
			log.Fatal(err)
		}

		certfile = filepath.Join(cnDir, "cert.pem")
		fallthrough
	default:
		verbose("output to %s", certfile)
		out, err = os.Create(certfile)
		if err != nil {
			log.Fatal(err)
		}
		defer out.Close()
	}

	// Finally do the output
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

}
