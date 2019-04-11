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
	CNs = CNList{}
	opt = struct {
		Certbase  string
		SSLFile   string
		Connect   string
		ServerCN  string
		Outfile   string
		CNfile    string
		Verbose   bool
		OutFormat Format
	}{OutFormat: FORMAT}
	verbose func(string, ...interface{})
)

func main() {

	urlBase := "https://" + opt.Connect

	// ohoh, even in Go we can write unreadable code, just to
	// avoid having tons of semiglobal variables
	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: func() *tls.Config {
			cert, err := tls.LoadX509KeyPair(opt.SSLFile, opt.SSLFile)
			if err != nil {
				log.Fatal(err)
			}
			pool, err := CertPool(opt.SSLFile)
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

	// build a list with required items, depending on the required
	// output format
	var items = func() []string {
		switch opt.OutFormat {
		case FormatPEM:
			return []string{"cert", "chain", "fullchain", "privkey"}
		case FormatPKCS12:
			return []string{"pkcs12"}
		default:
			panic("unknown output format")
		}
	}()

	// If we've a CNs list, append them to the CNs
	err := CNs.AppendFromFile(opt.CNfile)
	if err != nil {
		log.Fatal(err)
	}

	for CN, _ := range CNs {
		verbose("Request %s: %s\n", CN, items)
		for _, item := range items {
			URL := urlBase + "/" + item + "/" + CN
			verbose("Getting %s", URL)

			resp, err := http.Get(URL)
			if err != nil {
				log.Fatal(err)
			}
			if resp.StatusCode != http.StatusOK {
				log.Printf("%s: Status %s\n", CN, resp.Status)
				continue
			}
			defer resp.Body.Close()

			// Ok, we can get something, so prepare the destination,
			var out io.WriteCloser
			switch certfile := opt.Outfile; certfile {
			case "-":
				out = os.Stdout
				verbose("output to STDOUT")
			case "": // store in the certbase directory structure
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

}
