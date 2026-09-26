package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
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
	assert.NotContains(t, w.Body.String(), "unknown-client",
		"HTTP body must not echo CN from the underlying open error")
	assert.NotContains(t, w.Body.String(), opt.ClientConfigDir,
		"HTTP body must not reveal the configured clients dir path")
}

func TestAuthz_InvalidCN(t *testing.T) {
	setupTestEnv(t)

	// A client cert whose CN fails validation (here: contains a /)
	// must yield 401, not 500. The CN never names a config file, so
	// returning a server-error code on this path leaks the
	// validation outcome and breaks ordinary auth handling.
	req := mockTLSRequest("GET", "/v1/privkey/example.com", "foo/bar")
	w := httptest.NewRecorder()
	ctx := make(context)

	err := authz(ctx, w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Body.String(), "foo/bar",
		"HTTP body must not echo attacker-controlled CN")
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

func TestServe_ListInvalidCN(t *testing.T) {
	setupTestEnv(t)

	// /v1/list goes through authn → serve, so an invalid CN
	// reaches cnList without prior validation. serve must map
	// the resulting ErrInvalidName to 401, not 500.
	req := mockTLSRequest("GET", "/v1/list", "foo/bar")
	w := httptest.NewRecorder()
	ctx := context{REMOTE: "foo/bar"}

	err := serve(ctx, w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Body.String(), "foo/bar",
		"HTTP body must not echo attacker-controlled CN")
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
	certbase, _ := setupTestEnv(t)

	for _, role := range []string{"cert", "chain", "fullchain", "privkey"} {
		t.Run(role, func(t *testing.T) {
			req := mockTLSRequest("GET", "/v1/"+role+"/nonexistent.com", "")
			w := httptest.NewRecorder()

			err := serve(make(context), w, req)
			require.Error(t, err)
			assert.Equal(t, http.StatusNotFound, w.Code)
			assert.Equal(t, "not found\n", w.Body.String(),
				"HTTP body must be opaque")
			assert.NotContains(t, w.Body.String(), certbase)
		})
	}
}

func TestServe_BundleMissingDomain(t *testing.T) {
	certbase, _ := setupTestEnv(t)

	req := mockTLSRequest("GET", "/v1/bundle/nonexistent.com?format=PKCS12&pass=x", "")
	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "not found\n", w.Body.String())
	assert.NotContains(t, w.Body.String(), certbase)
}

func TestServe_UnreadableIsOpaque500(t *testing.T) {
	certbase, _ := setupTestEnv(t)

	// A self-referencing symlink fails with ELOOP, which is not
	// fs.ErrNotExist: it must be a 500 with an opaque body.
	if runtime.GOOS == "windows" {
		t.Skip("symlink loop not portable")
	}

	dir := filepath.Join(certbase, "loop.example.com")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.Symlink("cert.pem", filepath.Join(dir, "cert.pem")))

	req := mockTLSRequest("GET", "/v1/cert/loop.example.com", "")
	w := httptest.NewRecorder()

	err := serve(make(context), w, req)
	require.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "internal error\n", w.Body.String())
	assert.NotContains(t, w.Body.String(), certbase)
}

func TestServe_BundlePass(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	domain := "example.com"
	dir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(dir, 0755))
	createTestCertAndKey(t, dir)

	tests := []struct {
		name  string
		query string
		code  int
	}{
		{"empty", "format=PKCS12&pass=", http.StatusOK},
		{"set", "format=PKCS12&pass=secret", http.StatusOK},
		{"legacy", "format=PKCS12&pass=secret&pkcs12-compat=legacy", http.StatusOK},
		{"invalid compat", "format=PKCS12&pass=secret&pkcs12-compat=bogus", http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mockTLSRequest("GET", "/v1/bundle/"+domain+"?"+tc.query, "")
			w := httptest.NewRecorder()

			_ = serve(make(context), w, req)
			assert.Equal(t, tc.code, w.Code, w.Body.String())
		})
	}
}

// TestServe_BundleWithoutPass pins the upgrade path: clients up to v1.21.0
// omit pass= without -passout. That is served with the empty password and
// logged as deprecated, without echoing any password.
func TestServe_BundleWithoutPass(t *testing.T) {
	certbase, _ := setupTestEnv(t)
	domain := "example.com"
	dir := filepath.Join(certbase, domain)
	require.NoError(t, os.MkdirAll(dir, 0755))
	createTestCertAndKey(t, dir)

	var logs bytes.Buffer

	orig := log.Writer()

	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(orig) })

	req := mockTLSRequest("GET", "/v1/bundle/"+domain+"?format=PKCS12", "myhost")
	w := httptest.NewRecorder()
	ctx := context{REMOTE: "myhost"}

	require.NoError(t, serve(ctx, w, req))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	key, cert, _, err := pkcs12.DecodeChain(w.Body.Bytes(), "")
	require.NoError(t, err, "must decode with the empty password")
	assert.NotNil(t, key)
	assert.Equal(t, "test.example.com", cert.Subject.CommonName)

	assert.Contains(t, logs.String(), "bundle request without pass=")
	assert.Contains(t, logs.String(), "cn=myhost")
	assert.Equal(t, 1, strings.Count(logs.String(), "bundle request without pass="), "warn once per request")
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

func TestCnList_RejectsUnsafeName(t *testing.T) {
	_, ccd := setupTestEnv(t)

	// Plant a file at clients/sub/inner so that a successful traversal
	// would return its contents instead of an error.
	require.NoError(t, os.MkdirAll(filepath.Join(ccd, "sub"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(ccd, "sub", "inner"),
		[]byte("leaked.example.com\n"), 0644))

	// Also plant a file literally named CON (skipped on Windows,
	// where the name addresses the console device) so a Linux-only
	// validator would still happily open it; the Windows reserved-name
	// guard must apply regardless of the host OS.
	if runtime.GOOS != "windows" {
		require.NoError(t, os.WriteFile(
			filepath.Join(ccd, "CON"),
			[]byte("device.example.com\n"), 0644))
	}

	cases := []struct {
		name string
		cn   string
	}{
		{"slash", "sub/inner"},
		{"backslash", `sub\inner`},
		{"nul", "with\x00nul"},
		{"empty", ""},
		{"leading_dot", ".hidden"},
		{"wildcard", "*"},
		{"win_con", "CON"},
		{"win_con_lower", "con"},
		{"win_con_mixed", "Con"},
		{"win_con_with_ext", "CON.txt"},
		{"win_nul", "NUL"},
		{"win_com1", "COM1"},
		{"win_lpt9", "LPT9"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := cnList(tc.cn)
			require.Error(t, err)

			if tc.cn != "" {
				require.NotContains(t, err.Error(), tc.cn,
					"error must not echo attacker-controlled CN")
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
	t.Run("authn", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := mockTLSRequest("GET", "/", "test-client")

		versionCheck(context{REMOTE: "test-client"}, w, req)
		assert.NotEmpty(t, w.Header().Get("x-version"))
	})

	t.Run("client cert", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := mockTLSRequest("GET", "/", "test-client")

		versionCheck(make(context), w, req)
		assert.NotEmpty(t, w.Header().Get("x-version"))
	})
}

func TestVersionCheck_AnonymousNoHeader(t *testing.T) {
	for _, req := range []*http.Request{
		httptest.NewRequest("GET", "/", nil),
		mockTLSRequest("GET", "/", ""),
	} {
		w := httptest.NewRecorder()

		versionCheck(make(context), w, req)
		assert.Empty(t, w.Header().Values("x-version"))
	}
}

func TestParseTLSVersion(t *testing.T) {
	v, err := parseTLSVersion("1.2")
	require.NoError(t, err)
	assert.Equal(t, uint16(tls.VersionTLS12), v)

	v, err = parseTLSVersion("1.3")
	require.NoError(t, err)
	assert.Equal(t, uint16(tls.VersionTLS13), v)

	for _, bad := range []string{"", "1.1", "1.0", "TLS1.2", "1.4"} {
		_, err := parseTLSVersion(bad)
		assert.Error(t, err, bad)
	}
}

func TestCheckDir(t *testing.T) {
	base := t.TempDir()

	file := filepath.Join(base, "afile")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	tests := []struct {
		name string
		dir  string
		must string
	}{
		{"existing directory", base, ""},
		{"missing", filepath.Join(base, "absent"), "does not exist"},
		{"a file", file, "not a directory"},
	}

	if runtime.GOOS != "windows" {
		link := filepath.Join(base, "link")
		require.NoError(t, os.Symlink(base, link))

		dangling := filepath.Join(base, "dangling")
		require.NoError(t, os.Symlink(filepath.Join(base, "absent"), dangling))

		tests = append(tests,
			struct {
				name string
				dir  string
				must string
			}{"symlink to a directory", link, ""},
			struct {
				name string
				dir  string
				must string
			}{"dangling symlink", dangling, "symlink pointing nowhere"},
		)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkDir(tc.dir)
			if tc.must == "" {
				assert.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.must)
			assert.Contains(t, err.Error(), tc.dir, "the message must name the path")
		})
	}
}

// The server only reads: an empty or read-only directory passes, an
// unreadable one does not.
func TestCheckDirPermissions(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		assert.NoError(t, checkDir(t.TempDir()))
	})

	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}

	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission bits")
	}

	t.Run("read-only directory", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "example.com"), []byte("x"), 0644))
		require.NoError(t, os.Chmod(dir, 0500))
		t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

		assert.NoError(t, checkDir(dir), "write access must not be required")
	})

	t.Run("unreadable directory", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0))
		t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

		err := checkDir(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), dir, "the message must name the path")
		assert.Contains(t, err.Error(), "is not readable")
	})

	t.Run("search but no read permission", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0100))
		t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

		err := checkDir(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not readable")
	})
}
