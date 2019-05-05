package cert

import (
	"bytes"
	. "cert-proxy/internal/shared"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type role string

var UseSymlink = runtime.GOOS != "windows"
var Force = false

const (
	RoleINVALID   role = ``
	RoleCRT       role = `CERT`
	RoleKEY       role = `KEY`
	RoleCHAIN     role = `CHAIN`
	RoleFULLCHAIN role = `FULLCHAIN`
	RoleBUNDLE    role = `BUNDLE`
)

type Format string

const (
	FormatINVALID Format = ``
	FormatPEM     Format = `PEM`
	FormatPKCS12  Format = `PKCS12`
)

// Each Format has a set of Files with specific Roles
var ROLES = map[Format][]role{
	FormatPEM:    {RoleCRT, RoleKEY, RoleCHAIN, RoleFULLCHAIN},
	FormatPKCS12: {RoleBUNDLE},
}

// Each Roles has a fixed set of Templates
type templates struct {
	remote, local, env *template.Template
}

var TEMPLATES = map[role]templates{
	RoleCRT: {
		remote: tt(`{{.Proxy}}/v1/cert/{{.Domain}}`),
		local:  tt(`{{.Domain}}/crt.pem`),
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
		remote: tt(`{{.Proxy}}/v1/bundle/{{.Domain}}?format=PKCS12{{with.Pass}}&pass={{.}}{{end}}`),
		local:  tt(`{{.Domain}}/bundle.p12`),
		env:    tt(`BUNDLEFILE={{.Local}}`),
	},
}

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
}

func NewReq(domain, remote, basedir, hook string, format Format, pass string) (Req, error) {
	var req = Req{domain: domain, hook: hook, env: []string{`DOMAIN=` + domain}}
	var ctx = templateContext{Domain: domain, Proxy: remote, Pass: pass}

	// This format may require RoleCRT, RoleKey, … or RoleBUNDLE
	for _, role := range ROLES[format] {
		if templates, ok := TEMPLATES[role]; !ok {
			panic("template for " + string(role) + " is missing")
		} else {

			var item = certItem{
				role:    role,
				local:   filepath.Join(basedir, mustExpand(templates.local, ctx)),
				private: role == RoleKEY || role == RoleBUNDLE,
			}

			// The env needs the expanded item.local (file name)
			ctx.Local = item.local
			item.env = mustExpand(templates.env, ctx)

			if r, err := http.NewRequest(`GET`, mustExpand(templates.remote, ctx), nil); err != nil {
				return Req{}, err
			} else {
				item.remote = r
			}

			req.items = append(req.items, item)
		}
	}

	return req, nil

}

type Mutex interface {
	Lock()
	Unlock()
}

// Execute is the Workhorse. It operats on a single "request", that is,
// on all its files
func (req *Req) Execute(mtx Mutex) error {

	// First request all the data we need, this can be done in parallel,
	// but we'll wait until all are done
	for i, item := range req.items {

		// The file may exist already, use its timestamp for i-m-s
		// header
		if !Force {
			if fi, err := os.Stat(item.local); err == nil {
				item.remote.Header.Set(`if-modified-since`, fi.ModTime().Format(http.TimeFormat))
			}
		}

		Verbose("Requesting %s ims:%s", item.remote.URL, item.remote.Header.Get(`if-modified-since`))

		resp, err := http.DefaultClient.Do(item.remote)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
		case http.StatusNotModified:
			Verbose(resp.Status)
			return nil
		default:
			return fmt.Errorf("%v: %v",
				item.remote.URL, resp.Status)
		}

		req.items[i].data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	}

	// Got it, now write the output
	//
	var infix = strconv.Itoa(int(time.Now().Unix()))
	var infixed = map[string]string{}

	for _, item := range req.items {
		Verbose("Write %s", item.local)

		infixed[item.local] = func(s string) string {
			i := strings.LastIndex(s, ".")
			if i < 0 {
				panic("should not happen")
			}
			return s[0:i] + `-` + infix + s[i:]
		}(item.local)

		// Write the file (optionally create the directory first)
		if err := writeFile(infixed[item.local], item.data); err != nil {
			return err
		}
	}

	// Ok, and now create the symlinks
	//
	for name, _ := range infixed {
		var err error
		if UseSymlink {
			os.Remove(name)
			err = os.Symlink(filepath.Base(infixed[name]), name)
		} else {
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
	if req.hook != "" {
		Verbose("Hook %s for %s", req.hook, req.domain)

		var cmd = exec.Cmd{
			Path: req.hook,
			Env:  append(os.Environ(), req.env...),
			Args: []string{
				0: req.hook,
				1: `deploy_cert`,
				2: req.domain,
				//3..6: to be filled below
				7: fmt.Sprint(time.Now().Unix()),
			},
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		for _, i := range req.items {
			cmd.Env = append(cmd.Env, i.env)
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
		mtx.Lock()
		defer mtx.Unlock()
		return cmd.Run()
	} else {
		return nil
	}
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

func writeFile(name string, data []byte) error {

	if err := Mkdir(filepath.Dir(name)); err != nil {
		return err
	}

	// We do not use ioutil.WriteFile, as this would overwrite
	// an existing file
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}
