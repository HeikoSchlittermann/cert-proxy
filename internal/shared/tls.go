package shared

import (
	"crypto/tls"
	"crypto/x509"
)

func TLSClientConfig(sslFile string, config *tls.Config) (*tls.Config, error) {
	CAs, err := TLSConfig(sslFile, config)
	if err != nil {
		return config, err
	}
	config.RootCAs = CAs
	return config, err
}

func TLSServerConfig(sslFile string, config *tls.Config) (*tls.Config, error) {
	CAs, err := TLSConfig(sslFile, config)
	if err != nil {
		return config, err
	}
	config.ClientCAs = CAs
	return config, err
}

func TLSConfig(sslFile string, config *tls.Config) (pool *x509.CertPool, err error) {

	cert, err := tls.LoadX509KeyPair(sslFile, sslFile)
	if err != nil {
		return
	}

	pool, err = CertPool(sslFile)
	if err != nil {
		return
	}

    config.Certificates = []tls.Certificate{cert}
	return
}
