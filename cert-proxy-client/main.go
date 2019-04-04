package main

import (
	. "cert-proxy/shared"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	caFile, crtFile, keyFile string
	connect                  string
	serverCN                 string
)

func main() {

	cert, err := tls.LoadX509KeyPair(crtFile, keyFile)
	Check(err)

	tlsConfig := tls.Config{
		RootCAs:      CertPool(caFile),
		ServerName:   serverCN,
		Certificates: []tls.Certificate{cert},
	}

	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: &tlsConfig,
	}

	resp, err := http.Get("https://" + connect)
	Check(err)

	body, err := ioutil.ReadAll(resp.Body)
	Check(err)
	fmt.Println(string(body))

}
