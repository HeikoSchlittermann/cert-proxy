package main

import "flag"

func init() {
	flag.StringVar(&crtFile, "crt", "crt.pem", "certificate file")
	flag.StringVar(&keyFile, "key", "key.pem", "key file")
    flag.StringVar(&caFile, "ca", "ca.pem", "CA file")
	flag.StringVar(&serve, "serve", ":4433", "[host]:port for listener")
	flag.Parse()
}
