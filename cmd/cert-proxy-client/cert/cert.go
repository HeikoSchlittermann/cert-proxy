// Copyright 2019-2024 Heiko Schlittermann <hs@schlittermann.de>
// SPDX-License-Identifier: Apache-2.0

// Package cert builds and executes certificate-fetch requests against the
// cert-proxy server.
package cert

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"go.schlittermann.de/heiko/cert-proxy/internal/list"
	"go.schlittermann.de/heiko/cert-proxy/internal/program"
	"go.schlittermann.de/heiko/cert-proxy/internal/shared"
)

type role string

// UseSymlink controls whether downloaded files are exposed via symlinks.
// Defaults to true on Unix, false on Windows.
var UseSymlink = runtime.GOOS != "windows"

// Force makes Execute ignore If-Modified-Since and re-download every file.
var Force = false

// Roles identify the per-domain artifacts the client may fetch.
const (
	RoleCRT       role = `CERT`
	RoleKEY       role = `KEY`
	RoleCHAIN     role = `CHAIN`
	RoleFULLCHAIN role = `FULLCHAIN`
	RoleBUNDLE    role = `BUNDLE`
)

// Format selects the on-disk certificate representation.
type Format string

// Supported certificate formats.
const (
	FormatPEM    Format = `PEM`
	FormatPKCS12 Format = `PKCS12`
)

// ROLES maps each Format to the set of files (by Role) it requires.
var ROLES = map[Format][]role{
	FormatPEM:    {RoleCRT, RoleKEY, RoleCHAIN, RoleFULLCHAIN},
	FormatPKCS12: {RoleBUNDLE},
}

// Each Roles has a fixed set of Templates
type templates struct {
	remote, local, env *template.Template
}

// TEMPLATES maps each Role to the URL, file-path, and environment-variable
// templates used to fetch and place the artifact.
var TEMPLATES = map[role]templates{
	RoleCRT: {
		remote: tt(`{{.Proxy}}/v1/cert/{{.Domain}}`),
		local:  tt(`{{.Domain}}/cert.pem`),
		env:    tt(`CERTFILE={{.Local}}`),
	},
	RoleKEY: {
		remote: tt(`{{.Proxy}}/v1/privkey/{{.Domain}}`),
		local:  tt(`{{.Domain}}/privkey.pem`),
		env:    tt(`KEYFILE={{.Local}}`),
	},
	RoleCHAIN: {
		remote: tt(`{{.Proxy}}/v1/chain/{{.Domain}}`),
		local:  tt(`{{.Domain}}/chain.pem`),
		env:    tt(`CHAINFILE={{.Local}}`),
	},
	RoleFULLCHAIN: {
		remote: tt(`{{.Proxy}}/v1/fullchain/{{.Domain}}`),
		local:  tt(`{{.Domain}}/fullchain.pem`),
		env:    tt(`FULLCHAINFILE={{.Local}}`),
	},
	RoleBUNDLE: {
		remote: tt(`{{.Proxy}}/v1/bundle/{{.Domain}}?format=PKCS12{{with .Pass}}&pass={{.}}{{end}}{{with .Compat}}&pkcs12-compat={{.}}{{end}}`),
		local:  tt(`{{.Domain}}/bundle.pfx`), // Windows does not like .p12 here
		env:    tt(`BUNDLEFILE={{.Local}}`),
	},
}

// Req is a per-domain bundle of artifact fetches plus an optional hook.
type Req struct {
	domain string
	items  []certItem // depending on the Role…
	hook   string
	env    []string
}

type certItem struct {
	role       role
	remote     *http.Request
	local, env string
	private    bool
	data       []byte
}

type templateContext struct {
	Domain string
	Proxy  string
	Local  string
	Pass   string
	Compat string
}

// NewReq builds a Req for one domain, expanding the URL, file-path, and env
// templates for each Role required by the chosen Format.
func NewReq(domain, remote, basedir, hook string, format Format, pass, compat string) (Req, error) {
	if err := list.ValidateDomain(domain); err != nil {
		return Req{}, fmt.Errorf("invalid domain %q: %w", domain, err)
	}

	var (
		req = Req{domain: domain, hook: hook, env: []string{`DOMAIN=` + domain}}
		ctx = templateContext{Domain: domain, Proxy: remote, Pass: pass, Compat: compat}
	)

	// This format may require RoleCRT, RoleKey, … or RoleBUNDLE

	for _, role := range ROLES[format] {
		templates, ok := TEMPLATES[role]
		if !ok {
			panic("template for " + string(role) + " is missing")
		}

		var item = certItem{
			role:    role,
			local:   filepath.Join(basedir, mustExpand(templates.local, ctx)),
			private: role == RoleKEY || role == RoleBUNDLE,
		}

		// The env needs the expanded item.local (file name)
		ctx.Local = item.local
		item.env = mustExpand(templates.env, ctx)

		r, err := http.NewRequestWithContext(context.Background(), `GET`, mustExpand(templates.remote, ctx), nil)
		if err != nil {
			return Req{}, err
		}

		r.Header.Add(`x-version`, program.Version)
		item.remote = r

		req.items = append(req.items, item)
	}

	return req, nil
}

// Execute is the Workhorse. It operats on a single "request", that is,
// on all its files. The supplied context controls cancellation and deadlines
// for every HTTP request issued during this call.
func (req *Req) Execute(ctx context.Context, mtx sync.Locker) error {
	// First request all the data we need, this can be done in parallel,
	// but we'll wait until all are done
	for i, item := range req.items {
		// The file may exist already, use its timestamp for i-m-s
		// header
		if !Force {
			if fi, err := os.Stat(item.local); err == nil {
				item.remote.Header.Set(`if-modified-since`, fi.ModTime().UTC().Format(http.TimeFormat))
			}
		}

		shared.Verbose("Requesting %s ims:%s", item.remote.URL, item.remote.Header.Get(`if-modified-since`))

		resp, err := http.DefaultClient.Do(item.remote.WithContext(ctx))
		if err != nil {
			return err
		}
		defer resp.Body.Close() //nolint:errcheck

		switch resp.StatusCode {
		case http.StatusOK:
		case http.StatusNotModified:
			shared.Verbose("%s", resp.Status)
			continue
		default:
			return fmt.Errorf("%v: %v",
				item.remote.URL, resp.Status)
		}

		req.items[i].data, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	}

	// Got it, now write the output
	//
	var (
		infix   = strconv.FormatInt(time.Now().Unix(), 10)
		infixed = map[string]string{}
	)

	for _, item := range req.items {
		if len(item.data) == 0 {
			continue
		}

		shared.Verbose("Write %s", item.local)

		infixed[item.local] = func(s string) string {
			i := strings.LastIndex(s, ".")
			if i < 0 {
				panic("should not happen")
			}

			return s[0:i] + `-` + infix + s[i:]
		}(item.local)

		// Write the file (optionally create the directory first)
		if err := writeFile(infixed[item.local], item.data, item.private); err != nil {
			// Best-effort cleanup of orphaned infixed file on write failure
			_ = os.Remove(infixed[item.local])
			return err
		}
	}

	// Ok, and now create the symlinks
	//
	for name := range infixed {
		var err error

		if UseSymlink {
			err = replaceSymlink(name, filepath.Base(infixed[name]))
		} else {
			// On Windows, Rename(src, dst) replaces dst if it's a symlink or regular file.
			// On Unix, it follows the same atomic semantics. Safe even if dst pre-exists.
			err = os.Rename(infixed[name], name)
		}

		if err != nil {
			return err
		}
	}

	// Now it is time to run the hook
	// hook deploy_cert DOMAIN KEYFILE CERTFILE FULLCHAINFILE CHAINFILE TIMESTAMP
	// 0    1           2      3       4        5             6         7
	// These positional parameters appear as environment variables too,
	// plus an additional variable BUNDLEFILE
	if req.hook == "" {
		return nil
	}

	shared.Verbose("Hook %s for %s", req.hook, req.domain)

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	var cmd = exec.Cmd{
		Path:   req.hook,
		Env:    append(os.Environ(), req.env...),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	for _, i := range req.items {
		cmd.Env = append(cmd.Env, i.env)
	}

	if len(req.items) == 1 && req.items[0].role == RoleBUNDLE {
		cmd.Args = []string{req.hook, `deploy_cert`, req.domain, req.items[0].local, timestamp}
	} else {
		cmd.Args = []string{
			0: req.hook,
			1: `deploy_cert`,
			2: req.domain,
			7: timestamp,
		}

		for _, i := range req.items {
			switch i.role {
			case RoleKEY:
				cmd.Args[3] = i.local
			case RoleCRT:
				cmd.Args[4] = i.local
			case RoleFULLCHAIN:
				cmd.Args[5] = i.local
			case RoleCHAIN:
				cmd.Args[6] = i.local
			}
		}
	}

	mtx.Lock()
	defer mtx.Unlock()

	return cmd.Run()
}

func (req Req) String() string {
	return fmt.Sprintf("%s (%d items)", req.domain, len(req.items))
}

func tt(txt string) *template.Template {
	return template.Must(template.New(txt).Parse(txt))
}
func mustExpand(t *template.Template, ctx templateContext) string {
	var b = &bytes.Buffer{}
	if err := t.Execute(b, &ctx); err != nil {
		panic(err)
	}

	return b.String()
}

func writeFile(name string, data []byte, private bool) error {
	if err := shared.Mkdir(filepath.Dir(name)); err != nil {
		return err
	}

	var mode os.FileMode
	if private {
		mode = 0600
	} else {
		mode = 0644
	}

	// O_EXCL is unreliable on NFS; cert-proxy-client is not intended for NFS-backed storage.
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck

	if err := file.Chmod(mode); err != nil {
		return err
	}

	_, err = file.Write(data)

	return err
}

// replaceSymlink atomically replaces the symlink at name; avoids the TOCTOU window of Remove+Symlink.
func replaceSymlink(name, target string) error {
	tmp := name + ".tmp"

	_ = os.Remove(tmp)

	if err := os.Symlink(target, tmp); err != nil {
		return err
	}

	return os.Rename(tmp, name)
}
