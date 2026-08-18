# Repository instructions

This is the single source of agent guidance for this repository. The
tool-specific files (`.claude/CLAUDE.md`, `.codex/AGENTS.md`,
`.gemini/GEMINI.md`) are symlinks to this file.

## Project

Cert-proxy is a mutual-TLS certificate distribution system. A central server holds Let's Encrypt certificates; authenticated clients fetch them over HTTPS with X509 client certificates issued by a local CA. Authorization is per-domain, configured via files in `/etc/cert-proxy/clients/<cn>`.

## Build & Test

```bash
make all                    # build both binaries into build/
make test                   # go test ./...
make install                # install to /usr/local/bin
make install-client         # client only
make install-server         # server only
make install-ca             # install CA scripts to /etc/cert-proxy/ca
make update                 # go get -u ./... && go mod tidy
make clean

go generate ./...           # regenerate man/*.gz from man/*.md

go test ./...               # all tests
go test -run TestName ./cmd/cert-proxy-client/secret/  # single test
golangci-lint run ./...     # misspell, revive, wsl_v5

make test-packaging         # opt-in: install the .deb in a podman container
```

### Packaging test

`test/packaging/` is guarded by the `packaging` build tag, so `go test ./...`
never needs podman. It builds real packages with `gogogo pack` and installs them
in a throwaway container, because the issue #45 class of regression -- packaging
quietly stopping to ship a file or create a directory -- is invisible to unit
tests and to lintian.

- `CERT_PROXY_DEB_DIR=<dir>` reuses already-built packages instead of packing.
- Most cases run on `debian:trixie-slim` unchanged. The script removes
  `/etc/dpkg/dpkg.cfg.d/docker` first, or the slim image's `path-exclude` would
  discard `/usr/share/man` during install.
- The cases asserting the `ssl-cert` group and `/var/lib/cert-proxy/certs` need
  `systemd-sysusers`/`systemd-tmpfiles`, which the slim image lacks and the
  generated postinst tolerates silently. They use the image built by
  `make test-packaging-image` and skip while it is absent — provisioning it is
  an operator concern, since a restricted network needs proxy/TLS arguments
  (see that target).
- Every manual page must be packaged **exactly once**. Two packages shipping the
  same path cannot be co-installed; `TestBothPackagesCoInstall` pins that.

The Makefile only covers local development and installs. Release artifacts
(cross-built binaries, `.deb`) are built by gogogo from `.gogogo.conf`.

Cross-compilation: `.gogogo.conf` targets `windows/amd64`, `linux/amd64` and
`linux/arm64` with `CGO_ENABLED=0`.

Version comes from the Go build info (`runtime/debug.ReadBuildInfo`), i.e. from
the module version the build was stamped with; no `-ldflags -X` is involved.

## Architecture

### Server (`cmd/cert-proxy-server/`)

HTTP endpoints on port 4433 (TLS):

| Endpoint | Auth | Purpose |
|----------|------|---------|
| `GET /v1/list` | authn | List the domains *this client* is authorized for |
| `GET /v1/cert/<domain>` | none | Certificate (public) |
| `GET /v1/chain/<domain>` | none | CA chain (public) |
| `GET /v1/fullchain/<domain>` | none | Full chain (public) |
| `GET /v1/privkey/<domain>` | authz | Private key |
| `GET /v1/bundle/<domain>` | authz | PKCS12 bundle (generated on-the-fly, `?pkcs12-compat=legacy\|modern`) |

Request pipeline uses functional middleware composition: `use(authn, serve)` / `use(authz, serve)`.

Authorization reads `/etc/cert-proxy/clients/<cn>` — one domain per line, `#` comments.

Certificate files are stored as `<certbase>/<domain>/{cert,privkey,chain,fullchain}.pem`.

### Client (`cmd/cert-proxy-client/`)

1. Optionally fetches domain list from server
2. Dispatches domains to a worker pool (`worker/` package)
3. Each worker downloads cert artifacts with `If-Modified-Since` caching (`cert/` package)
4. Writes files atomically (timestamped files + symlinks on Unix, rename on Windows)
5. Executes per-domain hook, then a shared hook after all domains complete

Hook invocation: `<script> deploy_cert <DOMAIN> <KEYFILE> <CERTFILE> <FULLCHAIN> <CHAINFILE> <TIMESTAMP>`

Password/credential sources (`secret/` package): `pass:`, `file:`, `env:` URI schemes.

### Shared (`internal/`)

- **shared/tls.go** — `TLSClientConfig()` / `TLSServerConfig()` for mutual TLS setup
- **shared/certpool.go** — X509 CertPool construction from PEM files
- **program/** — Version/name/path (version from `runtime/debug` build info)
- **list/** — `OrderedStrings`, `UniqStrings`, `AddItemsFromFile()`

## Manual pages

- Canonical sources: `man/*.md` (go-md2man Markdown). Tracked generated pages:
  `man/*.[1-8].gz`.
- **Never edit the `.gz` files.** Regenerate with `go generate ./...`, which runs
  `man/gen.go` via the pinned `go tool go-md2man`.
- Pages: `cert-proxy-client(8)`, `cert-proxy-server(8)`, `cert-proxy-clients(5)`
  (the `clients/<cn>` and `-cnfile` line format), `cert-proxy(7)` (protocol,
  endpoints, on-disk layout).
- When flags, subcommands, config formats, environment variables, files,
  defaults, endpoints, or packaging change, update the affected Markdown page in
  the same change and regenerate.
- The pages are embedded in both binaries (`man/man.go`) and exposed as
  `cert-proxy-client man [<section>] [<page>]` and the same for the server. The
  embedded set, the `.gz` files and the `manpages:` lists in `.gogogo.conf` must
  stay in sync.
- Validate with `go generate ./... && git diff --exit-code man/` (determinism),
  `gzip -t man/*.gz`, `gzip -cd man/X.gz | groff -man -Tutf8 -ww`, and
  `go test ./man/`.
- Manual pages must stay `gzip -9`; Debian rejects weaker compression
  (lintian `poor-compression-in-manual-page`).

## Dependencies

Runtime: standard library plus `software.sslmate.com/src/go-pkcs12`. Tests use
`testify`. `go-md2man` is a build-only `tool` dependency for the manual pages.

## Deployment

Systemd units in `systemd/`:
- **Server:** `cert-proxy-server.service` (runs continuously)
- **Client:** `cert-proxy-client.timer` (daily, 2h random delay)

Both use `EnvironmentFile` for runtime options.

## CA Setup

Scripts in `CA/` create and manage the local certificate authority. Install with `make install-ca`, configure `vars.sh` from template, run `mkca` to initialize, `mkssl-pem` to issue certificates.
</content>
