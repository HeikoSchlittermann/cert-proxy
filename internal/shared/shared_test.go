package shared

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestSSLPEM(t *testing.T) string {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     []string{"localhost", "test-server"},
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caTemplate, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "ssl.pem")
	f, err := os.Create(path)
	require.NoError(t, err)

	defer f.Close() //nolint:errcheck

	_ = pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	_ = pem.Encode(f, &pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})
	_ = pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	return path
}

func TestMkdir_Creates(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "newdir")
	require.NoError(t, Mkdir(dir))

	fi, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, fi.IsDir())
}

func TestMkdir_Idempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "existing")
	_ = os.Mkdir(dir, 0755)

	assert.NoError(t, Mkdir(dir))
}

func TestMkdir_ConflictWithFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "afile")
	_ = os.WriteFile(path, []byte("x"), 0644)

	assert.Error(t, Mkdir(path))
}

func TestCertPool_ValidPEM(t *testing.T) {
	sslFile := generateTestSSLPEM(t)

	pool, err := CertPool(sslFile)
	require.NoError(t, err)
	assert.NotNil(t, pool)
}

func TestCertPool_MissingFile(t *testing.T) {
	_, err := CertPool("/nonexistent/file.pem")
	assert.Error(t, err)
}

func TestTLSConfig_PopulatesCertificates(t *testing.T) {
	sslFile := generateTestSSLPEM(t)
	config := &tls.Config{}

	pool, err := TLSConfig(sslFile, config)
	require.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Len(t, config.Certificates, 1)
}

func TestTLSConfig_InvalidFile(t *testing.T) {
	config := &tls.Config{}
	_, err := TLSConfig("/nonexistent.pem", config)
	assert.Error(t, err)
}

func TestTLSClientConfig_SetsRootCAs(t *testing.T) {
	sslFile := generateTestSSLPEM(t)
	config := &tls.Config{}

	result, err := TLSClientConfig(sslFile, config)
	require.NoError(t, err)
	assert.NotNil(t, result.RootCAs)
	assert.Len(t, result.Certificates, 1)
}

func TestTLSServerConfig_SetsClientCAs(t *testing.T) {
	sslFile := generateTestSSLPEM(t)
	config := &tls.Config{}

	result, err := TLSServerConfig(sslFile, config)
	require.NoError(t, err)
	assert.NotNil(t, result.ClientCAs)
	assert.Len(t, result.Certificates, 1)
}

func TestTLSClientConfig_InvalidFile(t *testing.T) {
	config := &tls.Config{}
	_, err := TLSClientConfig("/nonexistent.pem", config)
	assert.Error(t, err)
}

func TestTLSServerConfig_InvalidFile(t *testing.T) {
	config := &tls.Config{}
	_, err := TLSServerConfig("/nonexistent.pem", config)
	assert.Error(t, err)
}
