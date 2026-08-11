// Copyright 2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

//go:build packaging

// Package packaging_test installs the generated Debian packages in a
// throwaway container and inspects the result. It is opt-in because it needs
// podman, gogogo and (once) network access:
//
//	go test -tags packaging ./test/packaging/
//	make test-packaging
//
// It exists because the regression behind issue #45 -- the client package
// silently stopped creating /etc/cert-proxy/hook and /var/lib/cert-proxy/certs
// when packaging moved to gogogo -- is invisible to every unit test and to
// lintian. Only an actual install shows it.
package packaging_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// baseImage needs no preparation: dpkg is enough for every assertion
	// about the package payload itself.
	baseImage = "docker.io/library/debian:trixie-slim"

	// systemdImage additionally carries systemd-sysusers, systemd-tmpfiles
	// and man-db, which the generated postinst needs to create the
	// ssl-cert group and the certificate store. Built by
	// "make test-packaging-image".
	systemdImage = "localhost/cert-proxy-pkgtest"

	// unslim drops the slim image's dpkg exclusion of /usr/share/man, which
	// would otherwise discard the manual pages during install.
	unslim = "rm -f /etc/dpkg/dpkg.cfg.d/docker"
)

// debDir holds the directory with the built .deb files for the whole run.
var debDir string

func TestMain(m *testing.M) {
	for _, tool := range []string{"podman", "gogogo"} {
		if _, err := exec.LookPath(tool); err != nil {
			fmt.Fprintf(os.Stderr, "skip: %s not in PATH\n", tool)
			os.Exit(0)
		}
	}

	// Reusing a prepared directory keeps the edit/run cycle short; a plain
	// run builds the packages itself.
	if dir := os.Getenv("CERT_PROXY_DEB_DIR"); dir != "" {
		debDir = dir

		os.Exit(m.Run())
	}

	tmp, err := os.MkdirTemp("", "cert-proxy-packaging-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "temp dir: %v\n", err)
		os.Exit(1)
	}

	defer os.RemoveAll(tmp) //nolint:errcheck // best effort

	if debDir, err = buildPackages(tmp); err != nil {
		fmt.Fprintf(os.Stderr, "building packages: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// buildPackages runs gogogo from the module root and returns the directory
// holding the generated .deb files.
func buildPackages(outDir string) (string, error) {
	root, err := filepath.Abs("../..")
	if err != nil {
		return "", err
	}

	cmd := exec.Command("gogogo", "pack",
		"--no-network", "--allow-unclean", "--skip-doctor", "--out", outDir)
	cmd.Dir = root
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	// gogogo writes <outDir>/<version>/deb/*.deb.
	matches, err := filepath.Glob(filepath.Join(outDir, "*", "deb", "*.deb"))
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no .deb produced below %s", outDir)
	}

	return filepath.Dir(matches[0]), nil
}

// run executes a shell script inside image with the package directory bound
// at /debs, and returns its combined output.
func run(t *testing.T, image, script string) (string, error) {
	t.Helper()

	cmd := exec.Command("podman", "run", "--rm",
		"-v", debDir+":/debs:ro,z", image, "sh", "-euc", script)

	out, err := cmd.CombinedOutput()

	return string(out), err
}

// mustRun fails the test when the script does not succeed.
func mustRun(t *testing.T, image, script string) string {
	t.Helper()

	out, err := run(t, image, script)
	require.NoError(t, err, "script failed:\n%s", out)

	return out
}

// installBoth is the install step shared by most cases. Both packages are
// installed together, which is what caught them shipping the same manual
// page path and therefore refusing to co-install.
const installBoth = unslim + `
dpkg -i /debs/cert-proxy-client_*_amd64.deb /debs/cert-proxy-server_*_amd64.deb
`

// TestBothPackagesCoInstall is the regression test for two packages claiming
// the same file: dpkg refuses the second one.
func TestBothPackagesCoInstall(t *testing.T) {
	out := mustRun(t, baseImage, installBoth+`
dpkg -l cert-proxy-client cert-proxy-server | grep '^ii' | wc -l
`)
	assert.Contains(t, out, "2", "both packages must reach state ii")
	assert.NotContains(t, out, "trying to overwrite")
}

// TestIssue45Payload asserts the files whose disappearance was reported in
// issue #45. They are payload, so no systemd tooling is needed.
func TestIssue45Payload(t *testing.T) {
	out := mustRun(t, baseImage, installBoth+`
echo "hook=$(stat -c '%a %U %G' /etc/cert-proxy/hook)"
echo "etc_cert_proxy=$(test -d /etc/cert-proxy && echo yes)"
echo "default=$(stat -c '%a' /etc/default/cert-proxy-client)"
echo "tmpfiles=$(test -f /usr/lib/tmpfiles.d/cert-proxy-client.conf && echo yes)"
echo "sysusers=$(test -f /usr/lib/sysusers.d/cert-proxy-client.conf && echo yes)"
`)

	for _, want := range []string{
		"hook=644 root root",
		"etc_cert_proxy=yes",
		"default=644",
		"tmpfiles=yes",
		"sysusers=yes",
	} {
		assert.Contains(t, out, want)
	}
}

// TestConffilesRegistered guards the upgrade behaviour: an admin's edits to
// these files must survive, which only holds if dpkg knows they are conffiles.
func TestConffilesRegistered(t *testing.T) {
	out := mustRun(t, baseImage, installBoth+`
dpkg-query -W -f='${Conffiles}\n' cert-proxy-client cert-proxy-server
`)

	for _, want := range []string{
		"/etc/cert-proxy/hook",
		"/etc/default/cert-proxy-client",
		"/etc/default/cert-proxy-server",
		"/etc/cert-proxy/ca/lib/openssl.cnf",
	} {
		assert.Contains(t, out, want, "must be a registered conffile")
	}
}

// TestNoStrayUnitFiles pins the fix for the over-broad systemd glob, which
// used to install cert-proxy-*.default as if it were a unit.
func TestNoStrayUnitFiles(t *testing.T) {
	out := mustRun(t, baseImage, installBoth+`
ls /usr/lib/systemd/system/
`)
	assert.Contains(t, out, "cert-proxy-client.service")
	assert.Contains(t, out, "cert-proxy-client.timer")
	assert.Contains(t, out, "cert-proxy-server.service")
	assert.NotContains(t, out, ".default", "a .default file is not a unit")
}

// TestManualPagesInstalled checks the pages land where man(1) looks and that
// each page is packaged exactly once.
func TestManualPagesInstalled(t *testing.T) {
	out := mustRun(t, baseImage, installBoth+`
for p in man8/cert-proxy-client.8.gz man8/cert-proxy-server.8.gz \
         man5/cert-proxy-clients.5.gz man7/cert-proxy.7.gz; do
    echo "$p=$(stat -c '%a' /usr/share/man/$p)"
done
`)

	for _, want := range []string{
		"man8/cert-proxy-client.8.gz=644",
		"man8/cert-proxy-server.8.gz=644",
		"man5/cert-proxy-clients.5.gz=644",
		"man7/cert-proxy.7.gz=644",
	} {
		assert.Contains(t, out, want)
	}
}

// TestEmbeddedManualWorks checks the installed binaries can print their own
// pages without man-db present, which is what makes the shared pages
// reachable on a host that has only one of the two packages.
func TestEmbeddedManualWorks(t *testing.T) {
	out := mustRun(t, baseImage, installBoth+`
cert-proxy-client man   | head -2 | tail -1
cert-proxy-client man 5 | head -2 | tail -1
cert-proxy-server man   | head -2 | tail -1
cert-proxy-server man 7 | head -2 | tail -1
`)

	for _, want := range []string{
		".TH cert-proxy-client 8",
		".TH cert-proxy-clients 5",
		".TH cert-proxy-server 8",
		".TH cert-proxy 7",
	} {
		assert.Contains(t, out, want)
	}
}

func TestBinariesRun(t *testing.T) {
	out := mustRun(t, baseImage, installBoth+`
cert-proxy-client -version
cert-proxy-server -version
`)
	assert.Contains(t, out, "cert-proxy-client v")
	assert.Contains(t, out, "cert-proxy-server v")
}

// TestRemoveAndPurge checks both packages uninstall cleanly, including the
// statoverride bookkeeping the generated postrm performs.
func TestRemoveAndPurge(t *testing.T) {
	out := mustRun(t, baseImage, installBoth+`
dpkg -r cert-proxy-client cert-proxy-server
dpkg -P cert-proxy-client cert-proxy-server
echo "remaining=$(dpkg -l | grep -c cert-proxy || true)"
echo "binary=$(test -e /usr/bin/cert-proxy-client && echo present || echo gone)"
echo "conffile=$(test -e /etc/cert-proxy/hook && echo present || echo gone)"
`)
	assert.Contains(t, out, "remaining=0")
	assert.Contains(t, out, "binary=gone")
	assert.Contains(t, out, "conffile=gone")
}

// TestSystemdAssetsCreated is the other half of issue #45: the certificate
// store and the ssl-cert group, which the generated postinst delegates to
// systemd-sysusers and systemd-tmpfiles. Those are absent from the slim
// image, and the postinst tolerates that silently, so this case needs the
// prepared image.
func TestSystemdAssetsCreated(t *testing.T) {
	requireSystemdImage(t)

	out := mustRun(t, systemdImage, installBoth+`
echo "certs=$(stat -c '%a %U %G' /var/lib/cert-proxy/certs)"
echo "group=$(getent group ssl-cert | cut -d: -f1)"
`)
	assert.Contains(t, out, "certs=750 root ssl-cert",
		"the store must match the mode the pre-gogogo statoverride set")
	assert.Contains(t, out, "group=ssl-cert")
}

// TestManualPageRenders reads an installed page through man(1), catching roff
// that survives generation but not rendering.
func TestManualPageRenders(t *testing.T) {
	requireSystemdImage(t)

	out := mustRun(t, systemdImage, installBoth+`
MANPAGER=cat MANWIDTH=80 man 8 cert-proxy-client | head -4
`)
	assert.Contains(t, out, "cert-proxy-client(8)")
	assert.Contains(t, out, "fetch certificates from a cert-proxy server")
}

// requireSystemdImage skips unless the prepared image exists. Building it is
// deliberately not done here: it needs Debian archive access, and on a
// restricted network it needs proxy and TLS settings that belong in the
// operator's hands rather than in a test.
func requireSystemdImage(t *testing.T) {
	t.Helper()

	if exec.Command("podman", "image", "exists", systemdImage).Run() != nil {
		t.Skipf("%s missing; create it with \"make test-packaging-image\" "+
			"(see that target for proxy/TLS options on restricted networks)", systemdImage)
	}
}
