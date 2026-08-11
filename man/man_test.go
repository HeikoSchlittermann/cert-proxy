// Copyright 2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package man

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedPagesDecompress checks that every embedded page is intact and
// carries the .TH header for its own name and section.
func TestEmbeddedPagesDecompress(t *testing.T) {
	for _, p := range []Page{clientPage, serverPage, clientsPage, protocolPage} {
		t.Run(p.Name, func(t *testing.T) {
			roff, err := p.Roff()
			require.NoError(t, err)
			assert.Contains(t, string(roff), fmt.Sprintf(".TH %s %s ", p.Name, p.Section))
			assert.Contains(t, string(roff), p.Name+` \- `, "NAME section must describe the page")
		})
	}
}

func TestResolveDefaults(t *testing.T) {
	tests := []struct {
		name     string
		registry *Registry
		args     []string
		want     string
		section  string
	}{
		{"client: no args -> own page", ClientRegistry(), nil, "cert-proxy-client", "8"},
		{"server: no args -> own page", ServerRegistry(), nil, "cert-proxy-server", "8"},
		{"client: section 8 -> own page", ClientRegistry(), []string{"8"}, "cert-proxy-client", "8"},
		{"server: section 8 -> own page", ServerRegistry(), []string{"8"}, "cert-proxy-server", "8"},
		{"section 5 -> its only page", ClientRegistry(), []string{"5"}, "cert-proxy-clients", "5"},
		{"section 7 -> its only page", ClientRegistry(), []string{"7"}, "cert-proxy", "7"},
		{"explicit section and page", ClientRegistry(), []string{"8", "cert-proxy-server"}, "cert-proxy-server", "8"},
		{"bare name, other section", ClientRegistry(), []string{"cert-proxy-clients"}, "cert-proxy-clients", "5"},
		{"bare name, own section", ServerRegistry(), []string{"cert-proxy-client"}, "cert-proxy-client", "8"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := tc.registry.Resolve(tc.args)
			require.NoError(t, err)
			assert.Equal(t, tc.want, p.Name)
			assert.Equal(t, tc.section, p.Section)
		})
	}
}

// TestResolveBareNameLowestSectionWins documents the man(1) tiebreak: a name
// registered in more than one section resolves to the lowest section.
func TestResolveBareNameLowestSectionWins(t *testing.T) {
	r := newRegistry("8",
		Page{Name: "dup", Section: "8", file: clientPage.file},
		Page{Name: "dup", Section: "5", file: clientsPage.file},
	)

	p, err := r.Resolve([]string{"dup"})
	require.NoError(t, err)
	assert.Equal(t, "5", p.Section, "lowest section must win, as man(1) does")

	p, err = r.Resolve([]string{"8", "dup"})
	require.NoError(t, err)
	assert.Equal(t, "8", p.Section, "an explicit section still reaches the other page")
}

func TestResolveErrors(t *testing.T) {
	r := ClientRegistry()

	tests := []struct {
		name string
		args []string
		must string
	}{
		{"unknown section", []string{"3"}, `unknown section "3"`},
		{"unknown page in valid section", []string{"8", "nope"}, `unknown page "nope" in section 8`},
		{"unknown bare name", []string{"nope"}, `unknown section or page "nope"`},
		{"too many arguments", []string{"8", "cert-proxy-client", "extra"}, "too many arguments"},
		{"option-like argument", []string{"-x"}, `unknown section or page "-x"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := r.Resolve(tc.args)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.must)
		})
	}
}

// TestResolveErrorsListAlternatives checks the errors name what was available,
// not just what failed.
func TestResolveErrorsListAlternatives(t *testing.T) {
	r := ClientRegistry()

	_, err := r.Resolve([]string{"3"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "available sections: 5, 7, 8")

	_, err = r.Resolve([]string{"nope"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cert-proxy-clients(5)")
}

// TestDisplayNonTerminalWritesRoff asserts the piped case writes roff and
// never starts a viewer.
func TestDisplayNonTerminalWritesRoff(t *testing.T) {
	var out, errOut bytes.Buffer

	viewerStarted := false
	d := &Displayer{
		Out:        &out,
		Err:        &errOut,
		IsTerminal: func() bool { return false },
		LookPath:   func(string) (string, error) { return "/usr/bin/man", nil },
		Run:        func(string, []byte) error { viewerStarted = true; return nil },
	}

	require.NoError(t, d.Display(clientPage))
	assert.False(t, viewerStarted, "the viewer must not run when stdout is not a terminal")
	assert.Contains(t, out.String(), ".TH cert-proxy-client 8 ")
	assert.Empty(t, errOut.String())
}

// TestDisplayTerminalUsesViewer asserts the terminal case hands the roff to
// the viewer and does not also write it to stdout.
func TestDisplayTerminalUsesViewer(t *testing.T) {
	var out, errOut bytes.Buffer

	var handed []byte

	d := &Displayer{
		Out:        &out,
		Err:        &errOut,
		IsTerminal: func() bool { return true },
		LookPath:   func(string) (string, error) { return "/usr/bin/man", nil },
		Run:        func(_ string, roff []byte) error { handed = roff; return nil },
	}

	require.NoError(t, d.Display(protocolPage))
	assert.Contains(t, string(handed), ".TH cert-proxy 7 ")
	assert.Empty(t, out.String(), "stdout must stay untouched when the viewer renders")
}

// TestDisplayMissingViewerFallsBack covers the documented fallback: warn on
// stderr, still emit roff, succeed.
func TestDisplayMissingViewerFallsBack(t *testing.T) {
	var out, errOut bytes.Buffer

	d := &Displayer{
		Out:        &out,
		Err:        &errOut,
		IsTerminal: func() bool { return true },
		LookPath:   func(string) (string, error) { return "", errors.New("not found") },
		Run:        func(string, []byte) error { t.Fatal("viewer must not run"); return nil },
	}

	require.NoError(t, d.Display(clientsPage))
	assert.Contains(t, errOut.String(), "not found")
	assert.Contains(t, out.String(), ".TH cert-proxy-clients 5 ")
}

func TestDisplayPropagatesFailures(t *testing.T) {
	t.Run("viewer failure", func(t *testing.T) {
		d := &Displayer{
			Out:        &bytes.Buffer{},
			Err:        &bytes.Buffer{},
			IsTerminal: func() bool { return true },
			LookPath:   func(string) (string, error) { return "/usr/bin/man", nil },
			Run:        func(string, []byte) error { return errors.New("viewer exploded") },
		}
		assert.ErrorContains(t, d.Display(clientPage), "viewer exploded")
	})

	t.Run("write failure", func(t *testing.T) {
		d := &Displayer{
			Out:        failingWriter{},
			Err:        &bytes.Buffer{},
			IsTerminal: func() bool { return false },
			LookPath:   func(string) (string, error) { return "/usr/bin/man", nil },
			Run:        func(string, []byte) error { return nil },
		}
		assert.ErrorContains(t, d.Display(clientPage), "broken pipe")
	})
}

// TestRoffReportsDecompressionFailure ensures corrupt embedded data is
// reported rather than panicking.
func TestRoffReportsDecompressionFailure(t *testing.T) {
	_, err := Page{Name: "bogus", Section: "9", file: "cert-proxy.7.md"}.Roff()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus(9)")
}

func TestSectionsAndPagesAreOrdered(t *testing.T) {
	r := ClientRegistry()
	assert.Equal(t, []string{"5", "7", "8"}, r.Sections(), "sections ascend, for the bare-name tiebreak")

	names := make([]string, 0, 2)
	for _, p := range r.Pages("8") {
		names = append(names, p.Name)
	}

	assert.Equal(t, []string{"cert-proxy-client", "cert-proxy-server"}, names,
		"the binary's own page must be first, it is the section default")
}

func TestIsSection(t *testing.T) {
	for _, s := range []string{"1", "5", "8", "9"} {
		assert.True(t, isSection(s), s)
	}

	for _, s := range []string{"", "0", "10", "a", "8x", "-1"} {
		assert.False(t, isSection(s), s)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }
