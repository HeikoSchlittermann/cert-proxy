package main

import (
	. "cert-proxy/internal/shared"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func serve(ctx context, w http.ResponseWriter, req *http.Request) error {

	Verbose("Serving url=%s ims=%s cn=%s\n", req.URL, req.Header.Get(`if-modified-since`), ctx[REMOTE])
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

	var content io.ReadSeeker
	var mtime time.Time
	var role, domain string

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
		if domains, err := cnList(ctx[REMOTE]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		} else {
			fmt.Fprintln(w, strings.Join(domains.Items(), "\n"))
			return nil
		}
	case `bundle`:
		if file, err := http.Dir(opt.Certbase).Open((filepath.Join(domain, role+ext))); err == nil {
			defer file.Close()
			content = file
			if fi, err := file.Stat(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			} else {
				mtime = fi.ModTime()
			}
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
		if file, err := http.Dir(opt.Certbase).Open(fn); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		} else {
			defer file.Close()
			content = file

			if fi, err := file.Stat(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			} else {
				mtime = fi.ModTime()
			}
		}
	default:
		err := fmt.Errorf("invalid request `%s`", role)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	http.ServeContent(w, req, domain, mtime, content)
	return nil

}
