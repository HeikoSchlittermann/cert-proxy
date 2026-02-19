// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

func serve(ctx context, w http.ResponseWriter, req *http.Request) error {
	shared.Verbose("Serving url=%v%s%s\n",
		req.URL,
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
	versionCheck(w, req)

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
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}

			mtime = fi.ModTime()
		} else if os.IsNotExist(err) {
			content, mtime, err = createPKCS12(opt.Certbase, domain, req.URL.Query().Get("pass"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
	case `cert`, `chain`, `fullchain`, `privkey`:
		fn := filepath.Join(domain, role+ext)

		file, err := http.Dir(opt.Certbase).Open(fn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		defer file.Close() //nolint:errcheck // fh is passed to content

		content = file

		fi, err := file.Stat()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
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
