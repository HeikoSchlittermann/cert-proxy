package main

import "flag"

func init() {
	flag.StringVar(&crtFile, "crt", "crt.pem", "client certificate file")
	flag.StringVar(&keyFile, "key", "key.pem", "client certificate key file")
	flag.StringVar(&caFile, "ca", "ca.pem", "client certificate key file")
	flag.StringVar(&connect, "connect", "localhost:4433", "address of cert proxy server")
	flag.StringVar(&serverCN, "server name", "cert-proxy", "CN of the server")
	flag.Parse()
}
