package shared

import (
	"crypto/tls"
	"crypto/x509"
)

func TLSClientConfig(sslFile string) (config *tls.Config, err error) {
	config, CAs, err := TLSConfig(sslFile)
	if err != nil {
		return
	}
	config.RootCAs = CAs
	return
}

func TLSServerConfig(sslFile string) (config *tls.Config, err error) {
	config, CAs, err := TLSConfig(sslFile)
	if err != nil {
		return
	}
	config.ClientCAs = CAs
	return
}

func TLSConfig(sslFile string) (config *tls.Config, pool *x509.CertPool, err error) {
	cert, err := tls.LoadX509KeyPair(sslFile, sslFile)
	if err != nil {
		return
	}

	pool, err = CertPool(sslFile)
	if err != nil {
		return
	}

	config = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	return
}
