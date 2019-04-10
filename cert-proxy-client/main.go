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

	for _, item := range []string{"cert", "chain", "fullchain", "privkey"} {
		URL := urlBase + "/" + item + "/" + CN
		verbose("Getting %s", URL)

		resp, err := http.Get(URL)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("Status %s\n", resp.Status)
			os.Exit(1)
		}
		defer resp.Body.Close()

		// Ok, we can get something, so prepare the destination,
		var out io.WriteCloser
		switch certfile := opt.Outfile; certfile {
		case "-":
			out = os.Stdout
			verbose("output to STDOUT")
		case "":    // store in the certbase directory structure
			cnDir := filepath.Join(opt.Certbase, CN)
			switch err := os.Mkdir(cnDir, 0777); err != nil {
			case os.IsExist(err):
				if stat, err := os.Stat(cnDir); err != nil {
					log.Fatal(err)
				} else if stat.IsDir() {
					break
				}
				fallthrough
			default:
				log.Fatal(err)
			}

			certfile = filepath.Join(cnDir, item+".pem")
			fallthrough
		default:
			verbose("Output to %s", certfile)
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

}
