package shared

import (
	"crypto/x509"
	"io/ioutil"
)

func CertPool(files ...string) (pool *x509.CertPool) {
	pool = x509.NewCertPool()
	for _, f := range files {
		pem, err := ioutil.ReadFile(f)
        Check(err)
		if !pool.AppendCertsFromPEM(pem) {
			panic("Can't append to ca cert pool")
		}
	}
	return
}
