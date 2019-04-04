package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	crtFile  string
	keyFile  string
	caFile   string
	server   string
	serverCN string
)

func init() {
	flag.StringVar(&crtFile, "crt", "crt.pem", "client certificate file")
	flag.StringVar(&keyFile, "key", "key.pem", "client certificate key file")
	flag.StringVar(&caFile, "ca", "ca.pem", "client certificate key file")
	flag.StringVar(&server, "server", "localhost:4433", "address of cert proxy server")
	flag.StringVar(&serverCN, "server name", "cert-proxy", "CN of the server")
	flag.Parse()
}

func rootCAs(files ...string) (pool *x509.CertPool) {
	pool = x509.NewCertPool()
	for _, f := range files {
		pem, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			panic("Can't append ca cert to pool")
		}
	}
	return
}

func check(err error) {
	if err == nil {
		return
	}
	log.Fatal(err)
}

func main() {

	cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
	check(err)

	tlsConfig := tls.Config{
		RootCAs:      rootCAs(caFile),
		ServerName:   serverCN,
		Certificates: []tls.Certificate{cert},
	}

	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: &tlsConfig,
	}

	resp, err := http.Get("https://" + server)
	check(err)

	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	fmt.Println(string(body))

}
