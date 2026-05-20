package main

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockTLSRequest(method, path, cn string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.TLS = &tls.ConnectionState{}

	if cn != "" {
		req.TLS.PeerCertificates = []*x509.Certificate{
			{Subject: pkix.Name{CommonName: cn}},
		}
	}

	return req
}

func setupTestEnv(t *testing.T) (certbase, ccd string) {
	t.Helper()
	certbase = t.TempDir()
	ccd = t.TempDir()

	origCertbase := opt.Certbase
	origCCD := opt.ClientConfigDir

	t.Cleanup(func() {
		opt.Certbase = origCertbase
		opt.ClientConfigDir = origCCD
	})

	opt.Certbase = certbase
	opt.ClientConfigDir = ccd

	return certbase, ccd
}

func createDomainFiles(t *testing.T, certbase, domain string) {
	t.Helper()

	dir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(dir, 0755))

	files := map[string]string{
		"cert.pem":      "---CERT---\n",
		"privkey.pem":   "---KEY---\n",
		"chain.pem":     "---CHAIN---\n",
		"fullchain.pem": "---FULLCHAIN---\n",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
	}
}

func createClientConfig(t *testing.T, ccd, cn string, domains []string) {
	t.Helper()

	content := strings.Join(domains, "\n") + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(ccd, cn), []byte(content), 0644))
}

func TestAuthn_NoCert(t *testing.T) {
	req := mockTLSRequest("GET", "/v1/list", "")
	w := httptest.NewRecorder()
	ctx := make(context)

	err := authn(ctx, w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthn_WithCert(t *testing.T) {
	req := mockTLSRequest("GET", "/v1/list", "test-client")
	w := httptest.NewRecorder()
	ctx := make(context)

	err := authn(ctx, w, req)
	require.NoError(t, err)
	assert.Equal(t, "test-client", ctx[REMOTE])
}

func TestAuthz_NoCert(t *testing.T) {
	_, ccd := setupTestEnv(t)
	createClientConfig(t, ccd, "test-client", []string{"example.com"})

	req := mockTLSRequest("GET", "/v1/privkey/example.com", "")
	w := httptest.NewRecorder()
	ctx := make(context)

	err := authz(ctx, w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthz_NoConfigFile(t *testing.T) {
	setupTestEnv(t)

	req := mockTLSRequest("GET", "/v1/privkey/example.com", "unknown-client")
	w := httptest.NewRecorder()
	ctx := make(context)

	err := authz(ctx, w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthz_NotAuthorized(t *testing.T) {
	_, ccd := setupTestEnv(t)
	createClientConfig(t, ccd, "test-client", []string{"allowed.com"})

	req := mockTLSRequest("GET", "/v1/privkey/forbidden.com", "test-client")
	w := httptest.NewRecorder()
	ctx := make(context)

	err := authz(ctx, w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthz_Authorized(t *testing.T) {
	_, ccd := setupTestEnv(t)
	createClientConfig(t, ccd, "test-client", []string{"example.com", "sub.example.com"})

	req := mockTLSRequest("GET", "/v1/privkey/example.com", "test-client")
	w := httptest.NewRecorder()
	ctx := make(context)

	err := authz(ctx, w, req)
	require.NoError(t, err)
	assert.Equal(t, "example.com", ctx[DOMAIN])
	assert.Equal(t, "test-client", ctx[REMOTE])
}

func TestAuthz_MalformedPath(t *testing.T) {
	_, ccd := setupTestEnv(t)
	createClientConfig(t, ccd, "test-client", []string{"example.com"})

	req := mockTLSRequest("GET", "/v1/privkey", "test-client")
	w := httptest.NewRecorder()
	ctx := make(context)

	err := authz(ctx, w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServe_Cert(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	createDomainFiles(t, certbase, "example.com")

	req := mockTLSRequest("GET", "/v1/cert/example.com", "")
	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "---CERT---\n", w.Body.String())
}

func TestServe_Chain(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	createDomainFiles(t, certbase, "example.com")

	req := mockTLSRequest("GET", "/v1/chain/example.com", "")
	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.NoError(t, err)
	assert.Equal(t, "---CHAIN---\n", w.Body.String())
}

func TestServe_Fullchain(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	createDomainFiles(t, certbase, "example.com")

	req := mockTLSRequest("GET", "/v1/fullchain/example.com", "")
	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.NoError(t, err)
	assert.Equal(t, "---FULLCHAIN---\n", w.Body.String())
}

func TestServe_Privkey(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	createDomainFiles(t, certbase, "example.com")

	req := mockTLSRequest("GET", "/v1/privkey/example.com", "")
	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.NoError(t, err)
	assert.Equal(t, "---KEY---\n", w.Body.String())
}

func TestServe_List(t *testing.T) {
	_, ccd := setupTestEnv(t)
	createClientConfig(t, ccd, "test-client", []string{"example.com", "sub.example.com"})

	req := mockTLSRequest("GET", "/v1/list", "test-client")
	w := httptest.NewRecorder()
	ctx := context{REMOTE: "test-client"}

	err := serve(ctx, w, req)
	require.NoError(t, err)

	body := w.Body.String()
	assert.Contains(t, body, "example.com")
	assert.Contains(t, body, "sub.example.com")
}

func TestServe_NotModified(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	createDomainFiles(t, certbase, "example.com")

	req := mockTLSRequest("GET", "/v1/cert/example.com", "")
	req.Header.Set("If-Modified-Since", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))

	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotModified, w.Code)
}

func TestServe_InvalidFormat(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	createDomainFiles(t, certbase, "example.com")

	req := mockTLSRequest("GET", "/v1/cert/example.com?format=BOGUS", "")
	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServe_InvalidRole(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	createDomainFiles(t, certbase, "example.com")

	req := mockTLSRequest("GET", "/v1/bogus/example.com", "")
	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServe_MissingDomain(t *testing.T) {
	setupTestEnv(t)

	req := mockTLSRequest("GET", "/v1/cert/nonexistent.com", "")
	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServe_FormatPKCS12_Alias(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	domain := "example.com"
	dir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bundle.p12"), []byte("PKCS12DATA"), 0644))

	for _, format := range []string{"PKCS12", "PFX", "P12"} {
		t.Run(format, func(t *testing.T) {
			req := mockTLSRequest("GET", "/v1/bundle/"+domain+"?format="+format, "")
			w := httptest.NewRecorder()

			err := serve(make(context), w, req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "PKCS12DATA", w.Body.String())
		})
	}
}

func TestCnList_Valid(t *testing.T) {
	_, ccd := setupTestEnv(t)
	createClientConfig(t, ccd, "myhost", []string{"a.com", "b.com"})

	domains, err := cnList("myhost")
	require.NoError(t, err)
	assert.Len(t, domains.Items(), 2)
}

func TestCnList_Comments(t *testing.T) {
	_, ccd := setupTestEnv(t)
	content := "# comment\na.com\n# another\nb.com\n"
	require.NoError(t, os.WriteFile(filepath.Join(ccd, "myhost"), []byte(content), 0644))

	domains, err := cnList("myhost")
	require.NoError(t, err)
	assert.Len(t, domains.Items(), 2)
}

func TestCnList_Missing(t *testing.T) {
	setupTestEnv(t)

	_, err := cnList("nonexistent-client")
	require.Error(t, err)
}

// TestCnList_RejectsPathSeparator guards against a CN that points
// into a subdirectory of the clients config dir (issue #31).
func TestCnList_RejectsPathSeparator(t *testing.T) {
	_, ccd := setupTestEnv(t)

	// Plant a file at clients/sub/inner so that a successful traversal
	// would return its contents instead of an error.
	require.NoError(t, os.MkdirAll(filepath.Join(ccd, "sub"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(ccd, "sub", "inner"),
		[]byte("leaked.example.com\n"), 0644))

	for _, cn := range []string{
		"sub/inner",
		`sub\inner`,
		"with\x00nul",
		"",
		".hidden",
	} {
		t.Run(cn, func(t *testing.T) {
			_, err := cnList(cn)
			require.Error(t, err)
			if cn != "" {
				require.NotContains(t, err.Error(), cn,
					"error must not echo attacker-controlled CN (issue #29)")
			}
		})
	}
}

func TestUse_ChainStopsOnError(t *testing.T) {
	called := false

	failing := func(_ context, w http.ResponseWriter, _ *http.Request) error {
		http.Error(w, "fail", http.StatusForbidden)
		return http.ErrAbortHandler
	}

	second := func(_ context, _ http.ResponseWriter, _ *http.Request) error {
		called = true
		return nil
	}

	handler := use(failing, second)
	req := mockTLSRequest("GET", "/test", "")
	w := httptest.NewRecorder()
	handler(w, req)

	assert.False(t, called, "second handler should not run after error")
}

func TestUse_ChainCompletesOnSuccess(t *testing.T) {
	calls := 0

	h1 := func(_ context, _ http.ResponseWriter, _ *http.Request) error {
		calls++
		return nil
	}

	h2 := func(_ context, _ http.ResponseWriter, _ *http.Request) error {
		calls++
		return nil
	}

	handler := use(h1, h2)
	req := mockTLSRequest("GET", "/test", "")
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, 2, calls)
}

func TestVersionCheck_SetsHeader(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	versionCheck(w, req)
	assert.NotEmpty(t, w.Header().Get("x-version"))
}
