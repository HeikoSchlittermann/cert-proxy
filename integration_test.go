//go:build !skip_integration

package main_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	serverBin string
	clientBin string
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "certproxy-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	defer os.RemoveAll(tmpDir)

	serverBin = filepath.Join(tmpDir, "cert-proxy-server")
	clientBin = filepath.Join(tmpDir, "cert-proxy-client")

	if out, err := exec.Command("go", "build", "-o", serverBin, "./cmd/cert-proxy-server").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build server: %v\n%s\n", err, out)
		os.Exit(1)
	}

	if out, err := exec.Command("go", "build", "-o", clientBin, "./cmd/cert-proxy-client").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build client: %v\n%s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestServerBinary_Help(t *testing.T) {
	out, err := exec.Command(serverBin, "-help").CombinedOutput()
	require.NoError(t, err)

	output := string(out)
	assert.Contains(t, output, "certbase")
	assert.Contains(t, output, "sslfile")
	assert.Contains(t, output, "serve")
	assert.Contains(t, output, "ccd")
}

func TestServerBinary_Version(t *testing.T) {
	out, err := exec.Command(serverBin, "-version").CombinedOutput()
	require.NoError(t, err)
	assert.NotEmpty(t, strings.TrimSpace(string(out)))
}

func TestClientBinary_Help(t *testing.T) {
	out, err := exec.Command(clientBin, "-help").CombinedOutput()
	require.NoError(t, err)

	output := string(out)
	assert.Contains(t, output, "connect")
	assert.Contains(t, output, "sslfile")
	assert.Contains(t, output, "certbase")
	assert.Contains(t, output, "format")
	assert.Contains(t, output, "hook")
}

func TestClientBinary_Version(t *testing.T) {
	out, err := exec.Command(clientBin, "-version").CombinedOutput()
	require.NoError(t, err)
	assert.NotEmpty(t, strings.TrimSpace(string(out)))
}

type testPKI struct {
	caKey       *ecdsa.PrivateKey
	caCert      *x509.Certificate
	caCertDER   []byte
	serverSSL   string
	clientSSL   string
	certbaseDir string
	ccdDir      string
}

func setupPKI(t *testing.T) *testPKI {
	t.Helper()
	dir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Integration Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caCertDER)
	require.NoError(t, err)

	pki := &testPKI{
		caKey:       caKey,
		caCert:      caCert,
		caCertDER:   caCertDER,
		certbaseDir: filepath.Join(dir, "certbase"),
		ccdDir:      filepath.Join(dir, "clients"),
	}

	require.NoError(t, os.MkdirAll(pki.certbaseDir, 0755))
	require.NoError(t, os.MkdirAll(pki.ccdDir, 0755))

	pki.serverSSL = createSSLPEM(t, dir, "server", caKey, caCert, caCertDER, []string{"localhost", "127.0.0.1"})
	pki.clientSSL = createSSLPEM(t, dir, "test-client", caKey, caCert, caCertDER, nil)

	return pki
}

func createSSLPEM(t *testing.T, dir, cn string, caKey *ecdsa.PrivateKey, caCert *x509.Certificate, caCertDER []byte, sans []string) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     sans,
	}

	for _, s := range sans {
		if ip := net.ParseIP(s); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	path := filepath.Join(dir, cn+"-ssl.pem")
	f, err := os.Create(path)
	require.NoError(t, err)

	defer f.Close()

	pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	pem.Encode(f, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	return path
}

func freePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	return port
}

func startServer(t *testing.T, pki *testPKI, port int) {
	t.Helper()

	cmd := exec.Command(serverBin,
		"-sslfile", pki.serverSSL,
		"-serve", fmt.Sprintf("127.0.0.1:%d", port),
		"-certbase", pki.certbaseDir,
		"-ccd", pki.ccdDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())

	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()

			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("server did not start in time")
}

func createTestCerts(t *testing.T, certbase, domain string) {
	t.Helper()

	dir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(dir, 0755))

	files := map[string]string{
		"cert.pem":      "-----BEGIN CERTIFICATE-----\nTEST CERT\n-----END CERTIFICATE-----\n",
		"privkey.pem":   "-----BEGIN PRIVATE KEY-----\nTEST KEY\n-----END PRIVATE KEY-----\n",
		"chain.pem":     "-----BEGIN CERTIFICATE-----\nTEST CHAIN\n-----END CERTIFICATE-----\n",
		"fullchain.pem": "-----BEGIN CERTIFICATE-----\nTEST FULLCHAIN\n-----END CERTIFICATE-----\n",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
	}
}

func TestEndToEnd_PEM(t *testing.T) {
	pki := setupPKI(t)
	port := freePort(t)

	createTestCerts(t, pki.certbaseDir, "example.com")
	require.NoError(t, os.WriteFile(filepath.Join(pki.ccdDir, "test-client"), []byte("example.com\n"), 0644))

	startServer(t, pki, port)

	outDir := t.TempDir()
	out, err := exec.Command(clientBin,
		"-connect", fmt.Sprintf("https://127.0.0.1:%d", port),
		"-sslfile", pki.clientSSL,
		"-certbase", outDir,
		"-servername", "localhost",
		"-symlink=false",
		"-force",
	).CombinedOutput()

	require.NoError(t, err, "client failed: %s", string(out))

	for _, name := range []string{"cert.pem", "privkey.pem", "chain.pem", "fullchain.pem"} {
		path := filepath.Join(outDir, "example.com", name)
		_, err := os.Stat(path)
		assert.NoError(t, err, "expected file: %s", name)
	}
}

func TestEndToEnd_Force(t *testing.T) {
	pki := setupPKI(t)
	port := freePort(t)

	createTestCerts(t, pki.certbaseDir, "example.com")
	require.NoError(t, os.WriteFile(filepath.Join(pki.ccdDir, "test-client"), []byte("example.com\n"), 0644))

	startServer(t, pki, port)

	outDir := t.TempDir()
	args := []string{
		"-connect", fmt.Sprintf("https://127.0.0.1:%d", port),
		"-sslfile", pki.clientSSL,
		"-certbase", outDir,
		"-servername", "localhost",
		"-symlink=false",
		"-force",
	}

	out, err := exec.Command(clientBin, args...).CombinedOutput()
	require.NoError(t, err, "first run failed: %s", string(out))

	fi1, _ := os.Stat(filepath.Join(outDir, "example.com", "cert.pem"))

	time.Sleep(1100 * time.Millisecond)

	out, err = exec.Command(clientBin, args...).CombinedOutput()
	require.NoError(t, err, "second run failed: %s", string(out))

	fi2, _ := os.Stat(filepath.Join(outDir, "example.com", "cert.pem"))
	assert.True(t, fi2.ModTime().After(fi1.ModTime()), "file should be rewritten with -force")
}

func TestEndToEnd_AutoMode(t *testing.T) {
	pki := setupPKI(t)
	port := freePort(t)

	createTestCerts(t, pki.certbaseDir, "alpha.example.com")
	createTestCerts(t, pki.certbaseDir, "beta.example.com")
	require.NoError(t, os.WriteFile(filepath.Join(pki.ccdDir, "test-client"),
		[]byte("alpha.example.com\nbeta.example.com\n"), 0644))

	startServer(t, pki, port)

	outDir := t.TempDir()
	out, err := exec.Command(clientBin,
		"-connect", fmt.Sprintf("https://127.0.0.1:%d", port),
		"-sslfile", pki.clientSSL,
		"-certbase", outDir,
		"-servername", "localhost",
		"-symlink=false",
		"-force",
	).CombinedOutput()

	require.NoError(t, err, "client failed: %s", string(out))

	for _, domain := range []string{"alpha.example.com", "beta.example.com"} {
		path := filepath.Join(outDir, domain, "cert.pem")
		_, err := os.Stat(path)
		assert.NoError(t, err, "expected cert for %s", domain)
	}
}

func TestEndToEnd_ServerAPI_Direct(t *testing.T) {
	pki := setupPKI(t)
	port := freePort(t)

	createTestCerts(t, pki.certbaseDir, "example.com")
	require.NoError(t, os.WriteFile(filepath.Join(pki.ccdDir, "test-client"), []byte("example.com\n"), 0644))

	startServer(t, pki, port)

	tlsCert, err := tls.LoadX509KeyPair(pki.clientSSL, pki.clientSSL)
	require.NoError(t, err)

	caPEM, err := os.ReadFile(pki.clientSSL)
	require.NoError(t, err)

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caPEM)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{tlsCert},
				RootCAs:      caPool,
				ServerName:   "localhost",
			},
		},
	}

	baseURL := fmt.Sprintf("https://127.0.0.1:%d", port)

	t.Run("list", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/v1/list")
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "example.com")
	})

	t.Run("cert", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/v1/cert/example.com")
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "TEST CERT")
	})

	t.Run("privkey_authorized", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/v1/privkey/example.com")
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("privkey_unauthorized", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/v1/privkey/other.com")
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("x-version_header", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/v1/cert/example.com")
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.NotEmpty(t, resp.Header.Get("x-version"))
	})

	t.Run("if_modified_since_304", func(t *testing.T) {
		req, _ := http.NewRequest("GET", baseURL+"/v1/cert/example.com", nil)
		req.Header.Set("If-Modified-Since", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
		resp, err := client.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotModified, resp.StatusCode)
	})
}
