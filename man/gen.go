// Copyright 2026 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// Command gen converts every man/*.md source into the tracked
// <topic>.<section>.gz roff page beside it.
//
// Output is byte-for-byte reproducible: compress/gzip omits the source
// filename and sets MTIME=0 unless the caller sets Name/ModTime, which we
// deliberately do not. Each page is written to a temporary file in the
// destination directory and renamed into place, so a failing run can never
// leave a truncated tracked page behind.
package main

import (
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// manualPage matches the only thing this generator will touch:
// <topic>.<section>.md, the layout the packaging and the embed rely on.
// Anything else in the directory -- README.md, an editor scratch file -- is
// not a manual page and must not be turned into one.
var manualPage = regexp.MustCompile(`^[A-Za-z0-9._-]+\.[1-9]\.md$`)

func main() {
	candidates, err := filepath.Glob("*.md")
	if err != nil {
		fail(err)
	}

	var sources []string

	for _, c := range candidates {
		if manualPage.MatchString(filepath.Base(c)) {
			sources = append(sources, c)
		}
	}

	if len(sources) == 0 {
		fail(fmt.Errorf("no <topic>.<section>.md sources found in %s; run this through go generate", must(os.Getwd())))
	}

	for _, src := range sources {
		if err := generate(src); err != nil {
			fail(fmt.Errorf("%s: %w", src, err))
		}
	}
}

// generate renders src to roff, compresses it, and installs the result as
// <topic>.<section>.gz.
func generate(src string) error {
	dst := strings.TrimSuffix(src, ".md") + ".gz"

	roff, err := os.CreateTemp(".", ".roff-*")
	if err != nil {
		return err
	}

	defer os.Remove(roff.Name()) //nolint:errcheck // best effort cleanup

	if err := roff.Close(); err != nil {
		return err
	}

	// go tool keeps the converter pinned via the tool directive in go.mod.
	cmd := exec.Command("go", "tool", "go-md2man", "-in", src, "-out", roff.Name())
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go-md2man: %w", err)
	}

	return compress(roff.Name(), dst)
}

// compress gzips src into dst atomically.
func compress(src, dst string) error {
	plain, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if len(plain) == 0 {
		return fmt.Errorf("go-md2man produced no output")
	}

	tmp, err := os.CreateTemp(".", ".gz-*")
	if err != nil {
		return err
	}

	defer os.Remove(tmp.Name()) //nolint:errcheck // best effort cleanup

	// gzip -9: Debian rejects anything less for manual pages
	// (lintian: poor-compression-in-manual-page).
	zw, err := gzip.NewWriterLevel(tmp, gzip.BestCompression)
	if err != nil {
		return err
	}

	if _, err := zw.Write(plain); err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	// CreateTemp makes 0600 files; the installed page must be readable.
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), dst)
}

func must(s string, err error) string {
	if err != nil {
		fail(err)
	}

	return s
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "gen: %v\n", err)
	os.Exit(1)
}
