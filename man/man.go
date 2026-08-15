// Copyright 2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

// Package man carries the manual pages compiled into the cert-proxy
// binaries and implements the "man [<section>] [<page>]" subcommand.
//
// The canonical sources are the Markdown files in this directory; the
// embedded *.gz files are generated from them and must not be edited by
// hand. Regenerate with "go generate ./...".
package man

import (
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"sort"
	"strings"
)

//go:generate go run gen.go

// Command is the positional argument that selects the manual subcommand.
const Command = "man"

//go:embed cert-proxy-client.8.gz cert-proxy-server.8.gz cert-proxy-clients.5.gz cert-proxy.7.gz
var pages embed.FS

// pagesFS is the source Roff reads from. It is a variable so tests can supply
// deliberately damaged data without shipping a corrupt page inside the
// binaries.
var pagesFS fs.FS = pages

// Page is a single manual page embedded in the binary.
type Page struct {
	Name        string // topic, e.g. "cert-proxy.conf"
	Section     string // section digit, e.g. "8"
	Description string // one line, used for shell completion and errors
	file        string // name within the embedded FS
}

// Roff returns the decompressed roff source of the page.
func (p Page) Roff() ([]byte, error) {
	compressed, err := fs.ReadFile(pagesFS, p.file)
	if err != nil {
		return nil, fmt.Errorf("man: %s(%s): %w", p.Name, p.Section, err)
	}

	zr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("man: %s(%s): %w", p.Name, p.Section, err)
	}

	plain, err := io.ReadAll(zr)
	if err != nil {
		_ = zr.Close()

		return nil, fmt.Errorf("man: %s(%s): %w", p.Name, p.Section, err)
	}

	if err := zr.Close(); err != nil {
		return nil, fmt.Errorf("man: %s(%s): %w", p.Name, p.Section, err)
	}

	return plain, nil
}

// The pages shipped by both binaries.
var (
	clientPage   = Page{Name: "cert-proxy-client", Section: "8", Description: "cert-proxy-client(8) command manual", file: "cert-proxy-client.8.gz"}
	serverPage   = Page{Name: "cert-proxy-server", Section: "8", Description: "cert-proxy-server(8) command manual", file: "cert-proxy-server.8.gz"}
	clientsPage  = Page{Name: "cert-proxy-clients", Section: "5", Description: "cert-proxy-clients(5) authorization file format", file: "cert-proxy-clients.5.gz"}
	protocolPage = Page{Name: "cert-proxy", Section: "7", Description: "cert-proxy(7) protocol and layout overview", file: "cert-proxy.7.gz"}
)

// Registry holds the pages one binary offers, in per-section selection
// order: the first page registered in a section is that section's default.
type Registry struct {
	defaultSection string
	sections       map[string][]Page
}

// ClientRegistry is the page set of cert-proxy-client, defaulting to
// cert-proxy-client(8).
func ClientRegistry() *Registry {
	return newRegistry("8", clientPage, serverPage, clientsPage, protocolPage)
}

// ServerRegistry is the page set of cert-proxy-server, defaulting to
// cert-proxy-server(8).
func ServerRegistry() *Registry {
	return newRegistry("8", serverPage, clientPage, clientsPage, protocolPage)
}

func newRegistry(defaultSection string, list ...Page) *Registry {
	r := &Registry{defaultSection: defaultSection, sections: map[string][]Page{}}
	for _, p := range list {
		r.sections[p.Section] = append(r.sections[p.Section], p)
	}

	return r
}

// Sections returns the registered section digits in ascending order.
func (r *Registry) Sections() []string {
	out := make([]string, 0, len(r.sections))
	for s := range r.sections {
		out = append(out, s)
	}

	sort.Strings(out)

	return out
}

// Pages returns the pages of a section in selection order.
func (r *Registry) Pages(section string) []Page { return r.sections[section] }

// Resolve implements the "man [<section>] [<page>]" argument table.
func (r *Registry) Resolve(args []string) (Page, error) {
	switch len(args) {
	case 0:
		return r.page(r.defaultSection, "")
	case 1:
		if isSection(args[0]) {
			return r.page(args[0], "")
		}

		return r.byName(args[0])
	case 2:
		return r.page(args[0], args[1])
	default:
		return Page{}, fmt.Errorf("man: too many arguments; usage: man [<section>] [<page>]")
	}
}

// byName looks a bare page name up across all sections. Sections are scanned
// in ascending numeric order, so the lowest section wins when a name is
// registered more than once. (man(1) resolves such a collision by its
// configured SECTION order, which is not ascending; ascending is merely a
// stable rule that happens to agree with it for the sections shipped here.)
func (r *Registry) byName(name string) (Page, error) {
	for _, s := range r.Sections() {
		for _, p := range r.sections[s] {
			if p.Name == name {
				return p, nil
			}
		}
	}

	return Page{}, fmt.Errorf("man: unknown section or page %q; available pages: %s", name, strings.Join(r.names(), ", "))
}

// page returns the named page of a section; an empty name selects that
// section's default page.
func (r *Registry) page(section, name string) (Page, error) {
	list, ok := r.sections[section]
	if !ok {
		return Page{}, fmt.Errorf("man: unknown section %q; available sections: %s", section, strings.Join(r.Sections(), ", "))
	}

	if name == "" {
		return list[0], nil
	}

	for _, p := range list {
		if p.Name == name {
			return p, nil
		}
	}

	return Page{}, fmt.Errorf("man: unknown page %q in section %s; available: %s", name, section, strings.Join(sectionNames(list), ", "))
}

func (r *Registry) names() []string {
	var out []string
	for _, s := range r.Sections() {
		out = append(out, sectionNames(r.sections[s])...)
	}

	return out
}

func sectionNames(list []Page) []string {
	out := make([]string, 0, len(list))
	for _, p := range list {
		out = append(out, fmt.Sprintf("%s(%s)", p.Name, p.Section))
	}

	return out
}

// isSection reports whether s is a single manual section digit.
func isSection(s string) bool {
	return len(s) == 1 && s[0] >= '1' && s[0] <= '9'
}

// Displayer renders a resolved page. The seams exist so tests need no
// terminal and start no pager.
type Displayer struct {
	Out        io.Writer
	Err        io.Writer
	IsTerminal func() bool
	LookPath   func(string) (string, error)
	Run        func(viewer string, roff []byte) error
}

// NewDisplayer returns a Displayer wired to the real process.
func NewDisplayer() *Displayer {
	return &Displayer{
		Out:        os.Stdout,
		Err:        os.Stderr,
		IsTerminal: stdoutIsTerminal,
		LookPath:   exec.LookPath,
		Run:        runViewer,
	}
}

// Display writes the page: through man(1) when standard output is a
// terminal, as raw roff otherwise.
func (d *Displayer) Display(p Page) error {
	roff, err := p.Roff()
	if err != nil {
		return err
	}

	if !d.IsTerminal() {
		_, err := d.Out.Write(roff)

		return err
	}

	viewer, err := d.LookPath("man")
	if err != nil {
		_, _ = fmt.Fprintln(d.Err, "man: man(1) not found, writing raw roff")

		if _, err := d.Out.Write(roff); err != nil {
			return err
		}

		return nil
	}

	return d.Run(viewer, roff)
}

// runViewer feeds the roff to "man -l -", the man-db form for reading a
// local page from standard input.
func runViewer(viewer string, roff []byte) error {
	cmd := exec.Command(viewer, "-l", "-")
	cmd.Stdin = bytes.NewReader(roff)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Run resolves args and displays the selected page. It is the whole
// subcommand.
func Run(r *Registry, args []string) error {
	p, err := r.Resolve(args)
	if err != nil {
		return err
	}

	return NewDisplayer().Display(p)
}
