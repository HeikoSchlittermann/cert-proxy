// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.schlittermann.de/heiko/cert-proxy/internal/list"
	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

// fileError reports a failure to read certificate material to the client
// without revealing paths or other details: a missing file is 404, anything
// else 500. The full error is returned, so use() logs it.
func fileError(w http.ResponseWriter, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		http.Error(w, "not found", http.StatusNotFound)
	} else {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}

	return err
}

func serve(ctx context, w http.ResponseWriter, req *http.Request) error {
	shared.Verbose("Serving url=%v%s%s\n",
		redactedURL(req.URL),
		func() string {
			if s := req.Header.Get(`if-modified-since`); s != "" {
				return " ims=" + s
			}

			return ""
		}(),
		func() string {
			if s := ctx[REMOTE]; s != "" {
				return " cn=" + s
			}

			return ""
		}())
	versionCheck(ctx, w, req)

	var ext string

	switch format := strings.ToUpper(req.URL.Query().Get(`format`)); format {
	case `PEM`, ``:
		ext = `.pem`
	case `PKCS12`, `PFX`, `P12`:
		ext = `.p12`
	default:
		err := fmt.Errorf("invalid format `%s`", format)
		http.Error(w, err.Error(), http.StatusBadRequest)

		return err
	}

	var (
		content      io.ReadSeeker
		mtime        time.Time
		role, domain string
	)

	// [0]/[1]v1/[2]<role>/[3]<domain>

	switch parts := strings.Split(req.URL.Path, "/"); len(parts) {
	case 4:
		domain = parts[3]
		fallthrough
	case 3:
		role = parts[2]
	default:
		err := errors.New("required syntax: /v1/<req>[/<domain>]")
		http.Error(w, err.Error(), http.StatusBadRequest)

		return err
	}

	switch role {
	case `list`:
		domains, err := cnList(ctx[REMOTE])
		if err != nil {
			status := http.StatusInternalServerError
			body := err.Error()

			if os.IsNotExist(err) || errors.Is(err, list.ErrInvalidName) {
				status = http.StatusUnauthorized
				body = "unauthorized"
			}

			http.Error(w, body, status)

			return err
		}

		_, _ = fmt.Fprintln(w, strings.Join(domains.Items(), "\n"))

		return nil
	case `bundle`:
		if file, err := http.Dir(opt.Certbase).Open((filepath.Join(domain, role+ext))); err == nil {
			defer file.Close() //nolint:errcheck // we pass the FH to content, which should test whether there is a close, and check its error

			content = file

			fi, err := file.Stat()
			if err != nil {
				return fileError(w, err)
			}

			mtime = fi.ModTime()
		} else if os.IsNotExist(err) {
			// Clients up to v1.21.0 omit pass= when fetching without
			// -passout; accept that as the empty password, but say so.
			// Never log the password itself.
			if !req.URL.Query().Has("pass") {
				log.Printf("Remote %v cn=%s: bundle request without pass=; treating as empty password (deprecated, will be rejected in a future major release)", req.RemoteAddr, ctx[REMOTE])
			}

			content, mtime, err = createPKCS12(opt.Certbase, domain, req.URL.Query().Get("pass"), req.URL.Query().Get("pkcs12-compat"))
			if errors.Is(err, errInvalidCompat) {
				http.Error(w, errInvalidCompat.Error(), http.StatusBadRequest)
				return err
			}

			if err != nil {
				return fileError(w, err)
			}
		} else {
			return fileError(w, err)
		}
	case `cert`, `chain`, `fullchain`, `privkey`:
		fn := filepath.Join(domain, role+ext)

		file, err := http.Dir(opt.Certbase).Open(fn)
		if err != nil {
			return fileError(w, err)
		}
		defer file.Close() //nolint:errcheck // fh is passed to content

		content = file

		fi, err := file.Stat()
		if err != nil {
			return fileError(w, err)
		}

		mtime = fi.ModTime()
	default:
		err := fmt.Errorf("invalid request `%s`", role)
		http.Error(w, err.Error(), http.StatusBadRequest)

		return err
	}

	http.ServeContent(w, req, domain, mtime, content)

	return nil
}
