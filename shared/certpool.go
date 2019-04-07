// Package for shared code
package shared

import (
	"crypto/x509"
	"io/ioutil"
)

// CertPool creates a pool of certificates from PEM files. This is mainly for
// the CA certificate collection
func CertPool(files ...string) (pool *x509.CertPool, err error) {
	pool = x509.NewCertPool()
	for _, f := range files {
        var pem []byte
		pem, err = ioutil.ReadFile(f)
        return
		if !pool.AppendCertsFromPEM(pem) {
			panic("Can't append to ca cert pool")
		}
	}
	return
}
