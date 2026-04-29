package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestCertAndKey(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("openssl", "req",
		"-x509", "-newkey", "ec", "-pkeyopt", "ec_paramgen_curve:prime256v1",
		"-keyout", filepath.Join(dir, "privkey.pem"),
		"-out", filepath.Join(dir, "cert.pem"),
		"-days", "1",
		"-nodes",
		"-subj", "/CN=test.example.com")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "openssl req failed: %s", out)

	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "chain.pem"),
		mustReadFile(t, filepath.Join(dir, "cert.pem")),
		0644))
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestCreatePKCS12_Success(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}

	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	reader, mtime, err := createPKCS12(certbase, domain, "secret")
	require.NoError(t, err)
	assert.False(t, mtime.IsZero())
	assert.Greater(t, reader.Len(), 0, "PKCS12 output should not be empty")
}

func TestCreatePKCS12_EmptyPassword(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}

	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	reader, mtime, err := createPKCS12(certbase, domain, "")
	require.NoError(t, err)
	assert.False(t, mtime.IsZero())
	assert.Greater(t, reader.Len(), 0)
}

func TestCreatePKCS12_MissingCert(t *testing.T) {
	certbase := t.TempDir()
	domain := "nonexistent.example.com"

	_, _, err := createPKCS12(certbase, domain, "pass")
	assert.Error(t, err)
}

func TestCreatePKCS12_MissingKey(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}

	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(domainDir, "cert.pem"), []byte("---CERT---"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(domainDir, "chain.pem"), []byte("---CHAIN---"), 0644))

	_, _, err := createPKCS12(certbase, domain, "pass")
	assert.Error(t, err)
}

func TestCreatePKCS12_Mtime(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}

	certbase := t.TempDir()
	domain := "test.example.com"
	domainDir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	createTestCertAndKey(t, domainDir)

	certPath := filepath.Join(domainDir, "cert.pem")
	fi, err := os.Stat(certPath)
	require.NoError(t, err)

	_, mtime, err := createPKCS12(certbase, domain, "pass")
	require.NoError(t, err)
	assert.Equal(t, fi.ModTime(), mtime, "mtime should match cert.pem modification time")
}
