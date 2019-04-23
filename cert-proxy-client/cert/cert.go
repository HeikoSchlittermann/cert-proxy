package cert

import (
	"bytes"
	. "cert-proxy/internal/shared"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type Role string

const (
	RoleINVALID   Role = ``
	RoleCRT       Role = `CERT`
	RoleKEY       Role = `KEY`
	RoleCHAIN     Role = `CHAIN`
	RoleFULLCHAIN Role = `FULLCHAIN`
	RoleBUNDLE    Role = `BUNDLE`
)

type Format string

const (
	FormatINVALID Format = ``
	FormatPEM     Format = `PEM`
	FormatPKCS12  Format = `PKCS12`
)

// Each Format has a set of Files with specific Roles
var ROLES = map[Format][]Role{
	FormatPEM:    {RoleCRT, RoleKEY, RoleCHAIN, RoleFULLCHAIN},
	FormatPKCS12: {RoleBUNDLE},
}

// Each Roles has a fixed set of Templates
type templates struct {
	remote, local, env *template.Template
}

var TEMPLATES = map[Role]templates{
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
		remote: tt(`{{.Proxy}}/v1/bundle/{{.Domain}}`),
		local:  tt(`{{.Domain}}/bundle.p12`),
		env:    tt(`BUNDLEFILE={{.Local}}`),
	},
}

type Req struct {
	domain string
	items  []certItem // depending on the Role…
}

type certItem struct {
	remote     *http.Request
	local, env string
	private    bool
	data       []byte
}

type templateContext struct {
	Domain string
	Proxy  string
	Local  string
}

func NewReq(domain, remote string, basedir string, format Format) (Req, error) {
	var req = Req{domain: domain}
	var ctx = templateContext{Domain: domain, Proxy: remote}

	// This format may require RoleCRT, RoleKey, … or RoleBUNDLE
	for _, role := range ROLES[format] {
		if templates, ok := TEMPLATES[role]; !ok {
			panic("template for " + string(role) + " is missing")
		} else {

			var item = certItem{
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

// Execute is the Workhorse. It operats on a single "request", that is,
// on all its files
func (req *Req) Execute() error {

	// First request all the data we need, this can be done in parallel,
	// but we'll wait until all are done
	for i, item := range req.items {
		Verbose("Requesting %s", item.remote.URL)
		resp, err := http.DefaultClient.Do(item.remote)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
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
		err := func() error {
			if err := Mkdir(filepath.Dir(item.local)); err != nil {
				return err
			}

			file, err := os.Create(infixed[item.local])
			if err != nil {
				return err
			}
			defer file.Close()

			if n, err := file.Write(item.data); err != nil {
				return err
			} else {
				Verbose("Wrote %d", n)
			}
			return nil
		}()

		if err != nil {
			return err
		}
	}

	// Ok, and now create the symlinks
	for link, file := range infixed {
		os.Remove(link)
		if err := os.Symlink(filepath.Base(file), link); err != nil {
			return err
		}
	}

	// Now we've to care about the hooks
	return nil
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
