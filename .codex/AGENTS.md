# Codex instructions

First, read `AGENTS.md`, then use this file for Codex-specific guidance.

## Project

Cert-proxy is a mutual-TLS certificate distribution system. A central server holds Let's Encrypt certificates; authenticated clients fetch them over HTTPS with X509 client certificates issued by a local CA. Authorization is per-domain, configured via files in `/etc/cert-proxy/clients/<cn>`.

## Build & Test

```bash
make all                    # build both client and server
make test                   # go test ./...
make install                # install to /usr/local/bin
make install-client         # client only
make install-server         # server only
make install-ca             # install CA scripts to /etc/cert-proxy/ca
make update                 # go get -u ./... && go mod tidy
make clean

go test ./...               # all tests
go test -run TestName ./cmd/cert-proxy-client/secret/  # single test
golangci-lint run ./...     # misspell, revive, wsl_v5
```

Cross-compilation: `.gogogo.conf` targets `linux/amd64` and `linux/arm64` with `CGO_ENABLED=0`.

Version is injected via linker flag from `git describe --always --dirty=+`.

## Architecture

### Server (`cmd/cert-proxy-server/`)

HTTP endpoints on port 4433 (TLS):

| Endpoint | Auth | Purpose |
|----------|------|---------|
| `GET /v1/list` | authn | List all available domains |
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
- **program/** — Version/name/path (version set by linker)
- **list/** — `OrderedStrings`, `UniqStrings`, `AddItemsFromFile()`

## Dependencies

Pure standard library — no external Go dependencies.

## Deployment

Systemd units in `systemd/`:
- **Server:** `cert-proxy-server.service` (runs continuously)
- **Client:** `cert-proxy-client.timer` (daily, 2h random delay)

Both use `EnvironmentFile` for runtime options.

## CA Setup

Scripts in `CA/` create and manage the local certificate authority. Install with `make install-ca`, configure `vars.sh` from template, run `mkca` to initialize, `mkssl-pem` to issue certificates.
