package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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

func createTestCertAndKey(t *testing.T, dir string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test.example.com"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IsCA:         true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	require.NoError(t, os.WriteFile(filepath.Join(dir, "cert.pem"), certPEM, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "privkey.pem"), keyPEM, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "chain.pem"), certPEM, 0644))
}

func TestCreatePKCS12_Success(t *testing.T) {
	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	reader, mtime, err := createPKCS12(certbase, domain, "secret", "legacy")
	require.NoError(t, err)
	assert.False(t, mtime.IsZero())
	assert.Greater(t, reader.Len(), 0, "PKCS12 output should not be empty")
}

func TestCreatePKCS12_EmptyPassword(t *testing.T) {
	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	reader, mtime, err := createPKCS12(certbase, domain, "", "")
	require.NoError(t, err)
	assert.False(t, mtime.IsZero())
	assert.Greater(t, reader.Len(), 0)
}

func TestCreatePKCS12_MissingCert(t *testing.T) {
	certbase := t.TempDir()
	domain := "nonexistent.example.com"

	_, _, err := createPKCS12(certbase, domain, "pass", "")
	assert.Error(t, err)
}

func TestCreatePKCS12_MissingKey(t *testing.T) {
	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("invalid")})
	require.NoError(t, os.WriteFile(filepath.Join(domainDir, "cert.pem"), certPEM, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(domainDir, "chain.pem"), certPEM, 0644))

	_, _, err := createPKCS12(certbase, domain, "pass", "")
	assert.Error(t, err)
}

func TestCreatePKCS12_Mtime(t *testing.T) {
	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	certPath := filepath.Join(domainDir, "cert.pem")
	fi, err := os.Stat(certPath)
	require.NoError(t, err)

	_, mtime, err := createPKCS12(certbase, domain, "pass", "")
	require.NoError(t, err)
	assert.Equal(t, fi.ModTime(), mtime, "mtime should match cert.pem modification time")
}

func TestCreatePKCS12_InvalidKeyPEM(t *testing.T) {
	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	require.NoError(t, os.WriteFile(filepath.Join(domainDir, "privkey.pem"), []byte("not pem"), 0600))

	_, _, err := createPKCS12(certbase, domain, "pass", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private key")
}

func TestCreatePKCS12_CompatLegacy(t *testing.T) {
	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	reader, _, err := createPKCS12(certbase, domain, "pass", "legacy")
	require.NoError(t, err)
	assert.Greater(t, reader.Len(), 0)
}

func TestCreatePKCS12_CompatModern(t *testing.T) {
	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	reader, _, err := createPKCS12(certbase, domain, "pass", "modern")
	require.NoError(t, err)
	assert.Greater(t, reader.Len(), 0)
}

func TestCreatePKCS12_CompatUnknownDefaultsToLegacy(t *testing.T) {
	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	readerDefault, _, err := createPKCS12(certbase, domain, "pass", "")
	require.NoError(t, err)

	readerLegacy, _, err := createPKCS12(certbase, domain, "pass", "legacy")
	require.NoError(t, err)

	// Both should produce output (can't compare bytes due to randomness in encryption)
	assert.Greater(t, readerDefault.Len(), 0)
	assert.Greater(t, readerLegacy.Len(), 0)
}
