package cert

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/cert/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("CERT-CONTENT"))
	})
	mux.HandleFunc("/v1/privkey/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("KEY-CONTENT"))
	})
	mux.HandleFunc("/v1/chain/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("CHAIN-CONTENT"))
	})
	mux.HandleFunc("/v1/fullchain/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("FULLCHAIN-CONTENT"))
	})
	mux.HandleFunc("/v1/bundle/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("BUNDLE-CONTENT"))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv
}

func newMockServer304(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-Modified-Since") != "" {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		_, _ = w.Write([]byte("DATA"))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func newMockServer500(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	return srv
}

// saveGlobals must not be used with t.Parallel() — it mutates package-level
// globals (UseSymlink, Force). Awaits refactoring these into an options struct.
func saveGlobals(t *testing.T) {
	t.Helper()

	origSymlink := UseSymlink
	origForce := Force
	origTransport := http.DefaultClient.Transport

	t.Cleanup(func() {
		UseSymlink = origSymlink
		Force = origForce
		http.DefaultClient.Transport = origTransport
	})
}

func TestNewReq_PEM(t *testing.T) {
	req, err := NewReq("example.com", "https://proxy:4433", "/certs", "", FormatPEM, "", "")
	require.NoError(t, err)

	assert.Equal(t, "example.com", req.domain)
	assert.Len(t, req.items, 4)

	urls := make([]string, len(req.items))
	for i, item := range req.items {
		urls[i] = item.remote.URL.String()
	}

	assert.Contains(t, urls, "https://proxy:4433/v1/cert/example.com")
	assert.Contains(t, urls, "https://proxy:4433/v1/privkey/example.com")
	assert.Contains(t, urls, "https://proxy:4433/v1/chain/example.com")
	assert.Contains(t, urls, "https://proxy:4433/v1/fullchain/example.com")

	for _, item := range req.items {
		assert.True(t, strings.HasPrefix(item.local, "/certs/example.com/"))

		if item.role == RoleKEY {
			assert.True(t, item.private)
		}

		if item.role == RoleCRT || item.role == RoleCHAIN || item.role == RoleFULLCHAIN {
			assert.False(t, item.private)
		}
	}
}

func TestNewReq_PEM_EnvStrings(t *testing.T) {
	req, err := NewReq("example.com", "https://proxy:4433", "/certs", "", FormatPEM, "", "")
	require.NoError(t, err)

	envs := make(map[string]bool)
	for _, item := range req.items {
		envs[strings.SplitN(item.env, "=", 2)[0]] = true
	}

	assert.True(t, envs["CERTFILE"])
	assert.True(t, envs["KEYFILE"])
	assert.True(t, envs["CHAINFILE"])
	assert.True(t, envs["FULLCHAINFILE"])
}

func TestNewReq_PKCS12(t *testing.T) {
	req, err := NewReq("example.com", "https://proxy:4433", "/certs", "", FormatPKCS12, "secret", "")
	require.NoError(t, err)

	assert.Len(t, req.items, 1)
	item := req.items[0]
	assert.Equal(t, RoleBUNDLE, item.role)
	assert.True(t, item.private)
	assert.Contains(t, item.remote.URL.String(), "format=PKCS12")
	assert.Contains(t, item.remote.URL.String(), "pass=secret")
	assert.True(t, strings.HasSuffix(item.local, "bundle.pfx"))
	assert.True(t, strings.HasPrefix(item.env, "BUNDLEFILE="))
}

func TestNewReq_PKCS12_NoPass(t *testing.T) {
	req, err := NewReq("example.com", "https://proxy:4433", "/certs", "", FormatPKCS12, "", "")
	require.NoError(t, err)

	item := req.items[0]
	assert.NotContains(t, item.remote.URL.String(), "&pass=")
	assert.NotContains(t, item.remote.URL.String(), "pkcs12-compat=")
}

func TestNewReq_PKCS12_Compat(t *testing.T) {
	req, err := NewReq("example.com", "https://proxy:4433", "/certs", "", FormatPKCS12, "", "legacy")
	require.NoError(t, err)

	item := req.items[0]
	assert.Contains(t, item.remote.URL.String(), "pkcs12-compat=legacy")

	req, err = NewReq("example.com", "https://proxy:4433", "/certs", "", FormatPKCS12, "", "modern")
	require.NoError(t, err)

	item = req.items[0]
	assert.Contains(t, item.remote.URL.String(), "pkcs12-compat=modern")
}

func TestNewReq_Hook(t *testing.T) {
	req, err := NewReq("example.com", "https://proxy:4433", "/certs", "/usr/local/bin/hook.sh", FormatPEM, "", "")
	require.NoError(t, err)

	assert.Equal(t, "/usr/local/bin/hook.sh", req.hook)
}

func TestNewReq_DomainEnv(t *testing.T) {
	req, err := NewReq("example.com", "https://proxy:4433", "/certs", "", FormatPEM, "", "")
	require.NoError(t, err)

	assert.Contains(t, req.env, "DOMAIN=example.com")
}

func TestExecute_Download(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	for _, name := range []string{"cert.pem", "privkey.pem", "chain.pem", "fullchain.pem"} {
		path := filepath.Join(basedir, "example.com", name)
		_, err := os.Stat(path)
		assert.NoError(t, err, "file should exist: %s", name)
	}
}

func TestExecute_DownloadContent(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	data, err := os.ReadFile(filepath.Join(basedir, "example.com", "cert.pem"))
	require.NoError(t, err)
	assert.Equal(t, "CERT-CONTENT", string(data))
}

func TestExecute_Symlink(t *testing.T) {
	saveGlobals(t)

	UseSymlink = true
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	certPath := filepath.Join(basedir, "example.com", "cert.pem")
	fi, err := os.Lstat(certPath)
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "cert.pem should be a symlink")

	target, err := os.Readlink(certPath)
	require.NoError(t, err)
	assert.Contains(t, target, "cert-")
	assert.True(t, strings.HasSuffix(target, ".pem"))
}

func TestExecute_NoSymlink(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	certPath := filepath.Join(basedir, "example.com", "cert.pem")
	fi, err := os.Lstat(certPath)
	require.NoError(t, err)
	assert.Zero(t, fi.Mode()&os.ModeSymlink, "cert.pem should not be a symlink")
}

func TestExecute_NotModified(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = false

	srv := newMockServer304(t)
	basedir := t.TempDir()

	domainDir := filepath.Join(basedir, "example.com")
	require.NoError(t, os.MkdirAll(domainDir, 0755))

	for _, name := range []string{"cert.pem", "privkey.pem", "chain.pem", "fullchain.pem"} {
		require.NoError(t, os.WriteFile(filepath.Join(domainDir, name), []byte("OLD"), 0644))
	}

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	data, err := os.ReadFile(filepath.Join(domainDir, "cert.pem"))
	require.NoError(t, err)
	assert.Equal(t, "OLD", string(data), "file should not be overwritten on 304")
}

func TestExecute_ServerError(t *testing.T) {
	saveGlobals(t)

	Force = true

	srv := newMockServer500(t)
	basedir := t.TempDir()

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex

	err = req.Execute(context.Background(), &mtx)
	assert.Error(t, err)
}

func TestExecute_IfModifiedSince(t *testing.T) {
	saveGlobals(t)

	Force = false
	UseSymlink = false

	var imsCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-Modified-Since") != "" {
			imsCount++

			w.WriteHeader(http.StatusNotModified)

			return
		}

		_, _ = w.Write([]byte("DATA"))
	}))
	t.Cleanup(srv.Close)

	basedir := t.TempDir()
	domainDir := filepath.Join(basedir, "example.com")
	require.NoError(t, os.MkdirAll(domainDir, 0755))

	for _, name := range []string{"cert.pem", "privkey.pem", "chain.pem", "fullchain.pem"} {
		require.NoError(t, os.WriteFile(filepath.Join(domainDir, name), []byte("X"), 0644))
	}

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex

	_ = req.Execute(context.Background(), &mtx)

	assert.Equal(t, 4, imsCount, "all 4 items should send If-Modified-Since")
}

func TestExecute_Force(t *testing.T) {
	saveGlobals(t)

	Force = true

	var imsReceived string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		imsReceived = r.Header.Get("If-Modified-Since")

		_, _ = w.Write([]byte("NEW"))
	}))
	t.Cleanup(srv.Close)

	basedir := t.TempDir()
	domainDir := filepath.Join(basedir, "example.com")
	require.NoError(t, os.MkdirAll(domainDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(domainDir, "cert.pem"), []byte("OLD"), 0644))

	UseSymlink = false

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	assert.Empty(t, imsReceived, "Force should suppress If-Modified-Since")
}

func TestExecute_FilePermissions(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	certFi, _ := os.Stat(filepath.Join(basedir, "example.com", "cert.pem"))
	keyFi, _ := os.Stat(filepath.Join(basedir, "example.com", "privkey.pem"))

	certPerm := certFi.Mode().Perm()
	keyPerm := keyFi.Mode().Perm()

	assert.Equal(t, os.FileMode(0644), certPerm,
		"public file should be 0644, got %04o", certPerm)
	assert.Equal(t, os.FileMode(0600), keyPerm,
		"private file should be 0600, got %04o", keyPerm)
}

func TestExecute_FilePermissions_PreExisting(t *testing.T) {
	saveGlobals(t)

	UseSymlink = true
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	domainDir := filepath.Join(basedir, "example.com")
	require.NoError(t, os.MkdirAll(domainDir, 0755))

	// Pre-create a timestamped key file with overly broad permissions
	// to simulate a file left from a prior run with wrong perms.
	keyPath := filepath.Join(domainDir, "privkey-9999999999.pem")
	require.NoError(t, os.WriteFile(keyPath, []byte("old"), 0644))

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	// Read the symlink to find the actual timestamped file.
	symlinkPath := filepath.Join(domainDir, "privkey.pem")
	target, err := os.Readlink(symlinkPath)
	require.NoError(t, err)

	infixedPath := filepath.Join(domainDir, target)
	fi, err := os.Stat(infixedPath)
	require.NoError(t, err)

	assert.Equal(t, os.FileMode(0600), fi.Mode().Perm(),
		"private key must be 0600 even when pre-existing file had broader permissions")
}

func TestExecute_RenameExistingFile(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	domainDir := filepath.Join(basedir, "example.com")
	require.NoError(t, os.MkdirAll(domainDir, 0755))

	// Pre-create files at final paths (simulating previous rotation)
	certPath := filepath.Join(domainDir, "cert.pem")
	keyPath := filepath.Join(domainDir, "privkey.pem")

	require.NoError(t, os.WriteFile(certPath, []byte("old-cert"), 0644))
	require.NoError(t, os.WriteFile(keyPath, []byte("old-key"), 0644))

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	// Verify files were replaced with new content
	certData, err := os.ReadFile(certPath)
	require.NoError(t, err)
	assert.Equal(t, "CERT-CONTENT", string(certData), "cert should be replaced")

	keyData, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, "KEY-CONTENT", string(keyData), "privkey should be replaced")

	// Verify permissions are correct (0600 for keys)
	fi, err := os.Lstat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fi.Mode().Perm(),
		"private key must have 0600 permissions after os.Rename")
}

func TestExecute_PKCS12_Download(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPKCS12, "pass123", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	path := filepath.Join(basedir, "example.com", "bundle.pfx")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "BUNDLE-CONTENT", string(data))

	fi, _ := os.Stat(path)
	assert.Zero(t, fi.Mode().Perm()&0007, "bundle should have no other permissions")
}

func TestExecute_Hook_PEM(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	hookDir := t.TempDir()
	markerFile := filepath.Join(hookDir, "hook_called")
	hookScript := filepath.Join(hookDir, "hook.sh")

	script := "#!/bin/sh\necho \"$@\" > " + markerFile + "\nenv >> " + markerFile + "\n"
	require.NoError(t, os.WriteFile(hookScript, []byte(script), 0755))

	req, err := NewReq("example.com", srv.URL, basedir, hookScript, FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	data, err := os.ReadFile(markerFile)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, "deploy_cert")
	assert.Contains(t, output, "example.com")
	assert.Contains(t, output, "DOMAIN=example.com")
	assert.Contains(t, output, "KEYFILE=")
	assert.Contains(t, output, "CERTFILE=")
	assert.Contains(t, output, "FULLCHAINFILE=")
	assert.Contains(t, output, "CHAINFILE=")
}

func TestExecute_Hook_PKCS12(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	hookDir := t.TempDir()
	markerFile := filepath.Join(hookDir, "hook_called")
	hookScript := filepath.Join(hookDir, "hook.sh")

	script := "#!/bin/sh\necho \"$@\" > " + markerFile + "\nenv >> " + markerFile + "\n"
	require.NoError(t, os.WriteFile(hookScript, []byte(script), 0755))

	req, err := NewReq("example.com", srv.URL, basedir, hookScript, FormatPKCS12, "pass", "")
	require.NoError(t, err)

	var mtx sync.Mutex
	require.NoError(t, req.Execute(context.Background(), &mtx))

	data, err := os.ReadFile(markerFile)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, "deploy_cert")
	assert.Contains(t, output, "example.com")
	assert.Contains(t, output, "DOMAIN=example.com")
	assert.Contains(t, output, "BUNDLEFILE=")
}

func TestExecute_ContextCancellation(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-block:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(block) })

	basedir := t.TempDir()

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	var mtx sync.Mutex

	errCh := make(chan error, 1)

	go func() {
		errCh <- req.Execute(ctx, &mtx)
	}()

	cancel()

	select {
	case err := <-errCh:
		assert.Error(t, err, "Execute should return an error after context cancellation")
	case <-time.After(2 * time.Second):
		t.Fatal("Execute did not return after context cancellation")
	}
}

func TestExecute_CleanupOnWriteFileError(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = true

	srv := newMockServer(t)
	basedir := t.TempDir()

	req, err := NewReq("example.com", srv.URL, basedir, "", FormatPEM, "", "")
	require.NoError(t, err)

	domainDir := filepath.Join(basedir, "example.com")
	require.NoError(t, os.MkdirAll(domainDir, 0755))

	// Make the directory read-only to force writeFile to fail
	require.NoError(t, os.Chmod(domainDir, 0500))
	t.Cleanup(func() {
		// Restore write permissions so cleanup can work
		_ = os.Chmod(domainDir, 0755)
	})

	var mtx sync.Mutex

	err = req.Execute(context.Background(), &mtx)

	// Expect an error since the directory is read-only
	require.Error(t, err)

	// Restore write permissions to verify cleanup
	require.NoError(t, os.Chmod(domainDir, 0755))

	// Verify that no infixed files (with timestamps) were left behind
	entries, err := os.ReadDir(domainDir)
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		// Infixed files have format like: cert-<timestamp>.pem, privkey-<timestamp>.pem, etc.
		// Check that no such files exist by looking for the timestamp pattern
		if strings.Count(name, "-") >= 1 && (strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".pfx")) {
			t.Errorf("orphaned infixed file should have been cleaned up: %s", name)
		}
	}
}

func TestFormat_Set_Valid(t *testing.T) {
	var f Format

	require.NoError(t, f.Set("PEM"))
	assert.Equal(t, FormatPEM, f)

	require.NoError(t, f.Set("PKCS12"))
	assert.Equal(t, FormatPKCS12, f)

	require.NoError(t, f.Set("pem"))
	assert.Equal(t, FormatPEM, f)

	require.NoError(t, f.Set("pkcs12"))
	assert.Equal(t, FormatPKCS12, f)
}

func TestFormat_Set_Invalid(t *testing.T) {
	var f Format
	assert.Error(t, f.Set("BOGUS"))
}

func TestFormat_String(t *testing.T) {
	assert.Equal(t, "PEM", FormatPEM.String())
	assert.Equal(t, "PKCS12", FormatPKCS12.String())
	assert.Equal(t, "", Format("").String())
}

func TestReq_String(t *testing.T) {
	req, err := NewReq("example.com", "https://proxy:4433", "/certs", "", FormatPEM, "", "")
	require.NoError(t, err)

	s := req.String()
	assert.Contains(t, s, "example.com")
	assert.Contains(t, s, "4 items")
}

func TestReq_String_PKCS12(t *testing.T) {
	req, err := NewReq("example.com", "https://proxy:4433", "/certs", "", FormatPKCS12, "", "")
	require.NoError(t, err)

	s := req.String()
	assert.Contains(t, s, "1 items")
}

func TestNewReq_PathTraversal(t *testing.T) {
	traversals := []string{
		"../../etc/passwd",
		"../secret",
		"foo/../../etc/shadow",
	}

	for _, domain := range traversals {
		_, err := NewReq(domain, "https://proxy:4433", "/certs", "", FormatPEM, "", "")
		assert.Error(t, err, "domain %q must be rejected", domain)
	}
}

func TestNewReq_InvalidDomainChars(t *testing.T) {
	invalid := []string{
		"$(rm -rf /)",
		"foo;bar",
		"foo|bar",
		"foo bar",
		"`id`",
		"foo\x00bar",
		"foo\nbar",
	}

	for _, domain := range invalid {
		_, err := NewReq(domain, "https://proxy:4433", "/certs", "", FormatPEM, "", "")
		assert.Error(t, err, "domain %q must be rejected", domain)
	}
}

func TestNewReq_ValidDomains(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.example.com",
		"*.example.com",
		"a-b_c.example.com",
	}

	for _, domain := range valid {
		_, err := NewReq(domain, "https://proxy:4433", "/certs", "", FormatPEM, "", "")
		assert.NoError(t, err, "domain %q must be accepted", domain)
	}
}

func TestReplaceSymlink_Fresh(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "cert.pem")
	target := "cert-20240101.pem"

	require.NoError(t, replaceSymlink(name, target))

	got, err := os.Readlink(name)
	require.NoError(t, err)
	assert.Equal(t, target, got)

	_, err = os.Lstat(name + ".tmp")
	assert.True(t, os.IsNotExist(err), ".tmp file must not remain after replacement")
}

func TestReplaceSymlink_ExistingSymlink(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "cert.pem")
	oldTarget := "cert-old.pem"
	newTarget := "cert-new.pem"

	require.NoError(t, os.Symlink(oldTarget, name))
	require.NoError(t, replaceSymlink(name, newTarget))

	got, err := os.Readlink(name)
	require.NoError(t, err)
	assert.Equal(t, newTarget, got)

	_, err = os.Lstat(name + ".tmp")
	assert.True(t, os.IsNotExist(err), ".tmp file must not remain after replacement")
}

func TestWriteFile_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "privkey.pem")

	require.NoError(t, os.WriteFile(name, []byte("old"), 0644))

	err := writeFile(name, []byte("new"), true)
	require.Error(t, err)
	assert.True(t, os.IsExist(err), "expected an existing-file error, got %v", err)

	data, readErr := os.ReadFile(name)
	require.NoError(t, readErr)
	assert.Equal(t, "old", string(data))
}

func TestWriteFile_ExistingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privilege on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "victim.pem")
	name := filepath.Join(dir, "privkey.pem")

	require.NoError(t, os.WriteFile(target, []byte("SAFE"), 0644))
	require.NoError(t, os.Symlink(target, name))

	err := writeFile(name, []byte("new"), true)
	require.Error(t, err)
	assert.True(t, os.IsExist(err), "expected an existing-path error, got %v", err)

	data, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	assert.Equal(t, "SAFE", string(data))

	// Verify symlink itself was not disturbed
	linkTarget, linkErr := os.Readlink(name)
	require.NoError(t, linkErr, "symlink should still exist")
	assert.Equal(t, target, linkTarget, "symlink should still point to original target")
}

// TestExecute_DoesNotCreateCertbase pins a scope decision: the client creates
// the per-domain directory and nothing above it. Providing -certbase is the
// job of the package (a tmpfiles.d snippet) or of the administrator, so a
// missing store is an error rather than something to silently materialise
// with whatever ownership and mode the process happens to have.
func TestExecute_DoesNotCreateCertbase(t *testing.T) {
	saveGlobals(t)

	UseSymlink = false
	Force = false

	srv := newMockServer(t)

	base := t.TempDir()
	certbase := filepath.Join(base, "absent")

	req, err := NewReq("example.com", srv.URL, certbase, "", FormatPEM, "", "")
	require.NoError(t, err)

	var mtx sync.Mutex

	err = req.Execute(context.Background(), &mtx)
	require.Error(t, err, "a missing -certbase must not be created")
	assert.NoDirExists(t, certbase)
}
