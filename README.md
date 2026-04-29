# CERT-PROXY

Cert-proxy distributes TLS certificates from a central server to remote
systems that cannot obtain certificates from Let's Encrypt directly
(e.g., systems behind firewalls, without DNS access, or with limited
ACME challenge capabilities).

A separate ACME client (e.g., dehydrated) obtains and renews certificates
on the server. Cert-proxy then serves these certificates to authenticated
clients over mutual TLS.

```
      [ Let's Encrypt CA ]
         | |
         ^ |
         | v
         | |
       ACME client (dehydrated) ---> DNS
       cert-proxy-server
         | |
         ^ v
         | |
       cert-proxy-client
```

Authentication is mutual: the server verifies clients via X509 certificates
issued by a local CA, and clients verify the server's identity against the
same CA. Authorization is per-domain — the server controls which client may
access which private key.


## Building

Requires Go 1.26+.

```shell
git clone https://git.schlittermann.de/heiko/cert-proxy
cd cert-proxy
make install            # installs both binaries to /usr/local/bin/
```

Individual targets:

```shell
make install-server     # server only
make install-client     # client only
make install-ca         # CA scripts to /etc/cert-proxy/ca/
```

Cross-compilation for Windows:

```shell
GOOS=windows make install   # binaries in /usr/local/bin/windows_amd64/
```

Run tests and linter:

```shell
make test               # go test ./...
golangci-lint run ./... # misspell, revive, wsl_v5
```


## Server Setup

### 1. Set up the CA

The project includes a minimal CA for issuing mutual-auth certificates:

```shell
make install-ca
cd /etc/cert-proxy/ca
cp lib/vars.sh.example lib/vars.sh
vi lib/vars.sh              # set country, org, etc.
lib/mkca                    # initialize the CA (creates ca.crt + ca.key)
```

### 2. Create the server certificate

```shell
cd /etc/cert-proxy/ca
bin/mkssl-pem cert-proxy.example.com
cp cert-proxy.example.com-ssl.pem /etc/cert-proxy/server-ssl.pem
```

The `-ssl.pem` bundle contains the certificate, private key, and CA
certificate in a single PEM file.

### 3. Create the client configuration directory

```shell
install -d /etc/cert-proxy/clients
```

### 4. Start the service

```shell
systemctl enable $PWD/systemd/cert-proxy-server.service
systemctl start cert-proxy-server
```

The default systemd unit runs:

```
cert-proxy-server -verbose \
    -sslfile /etc/cert-proxy/server-ssl.pem \
    -certbase /var/lib/dehydrated/certs \
    -ccd /etc/cert-proxy/clients
```

### Server flags

| Flag | Default | Description |
|------|---------|-------------|
| `-sslfile` | `server-ssl.pem` | SSL auth file (crt+key+ca) PEM |
| `-serve` | `:4433` | Listener `[host]:port` |
| `-certbase` | `certs` | Base directory for certificates |
| `-ccd` | `clients` | Client configuration directory |
| `-verbose` | `false` | Verbose logging |


## Client Setup

### 1. Create a client certificate (on the server)

```shell
cd /etc/cert-proxy/ca
bin/mkssl-pem <client-name>
```

### 2. Authorize the client for specific domains

Create `/etc/cert-proxy/clients/<client-name>` on the server:

```
# one domain per line, comments allowed
example.com
sub.example.com
another.example.org
```

No server restart required when adding or changing client authorizations.

### 3. Deploy to the client system

Copy the following to the client:
- `<client-name>-ssl.pem` — client authentication bundle
- `cert-proxy-client` binary (or `.exe` for Windows)

### 4. Run the client

```shell
cert-proxy-client \
    -connect https://cert-proxy.example.com:4433 \
    -sslfile client-ssl.pem \
    -certbase /etc/ssl/certs \
    -verbose
```

For recurring operation, use the provided systemd timer:

```shell
systemctl enable $PWD/systemd/cert-proxy-client.timer
systemctl start cert-proxy-client.timer
```

The timer runs daily with a randomized delay of up to 2 hours.

### Client flags

| Flag | Default | Description |
|------|---------|-------------|
| `-connect` | `https://localhost:4433` | Server address |
| `-sslfile` | `client-ssl.pem` | SSL auth file (crt+key+ca) PEM |
| `-certbase` | `certs` | Base directory for downloaded certificates |
| `-format` | `PEM` (Linux), `PKCS12` (Windows) | Output format |
| `-auto` | `true` | Fetch all domains the server offers |
| `-cnfile` | | File with domain list (use `-` for stdin) |
| `-hook` | | Per-certificate hook script |
| `-shared-hook` | | Hook script called once after all certs are processed |
| `-passout` | | Password for PKCS12 bundles (see notation below) |
| `-servername` | | Expected server CN (default: hostname from `-connect`) |
| `-symlink` | `true` (Linux), `false` (Windows) | Use symlinks for atomic file updates |
| `-force` | `false` | Download even if server reports not-modified |
| `-jobs` | number of CPUs | Parallel download workers |

Domains can also be passed as positional arguments:

```shell
cert-proxy-client example.com sub.example.com
```


## API

All endpoints are served over mutual TLS on port 4433.

| Endpoint | Auth | Content |
|----------|------|---------|
| `GET /v1/list` | authn | List of all available domains |
| `GET /v1/cert/<domain>` | none | Certificate PEM |
| `GET /v1/chain/<domain>` | none | CA chain PEM |
| `GET /v1/fullchain/<domain>` | none | Certificate + chain PEM |
| `GET /v1/privkey/<domain>` | authz | Private key PEM |
| `GET /v1/bundle/<domain>?format=PKCS12` | authz | PKCS12 bundle (generated on-the-fly) |

- **authn** — valid client certificate required
- **authz** — client must be authorized for the requested domain

The client sends `If-Modified-Since` headers; the server responds with
`304 Not Modified` when the certificate has not changed.


## Hook Scripts

### Per-domain hook (`-hook`)

Called for each domain after its certificate files are written:

```
<script> deploy_cert <DOMAIN> <KEYFILE> <CERTFILE> <FULLCHAIN> <CHAINFILE> <TIMESTAMP>
```

For PKCS12 format:

```
<script> deploy_cert <DOMAIN> <BUNDLEFILE> <TIMESTAMP>
```

Environment variables set: `DOMAIN`, `KEYFILE`, `CERTFILE`, `FULLCHAINFILE`,
`CHAINFILE`, `BUNDLEFILE` (as applicable).

Hooks run sequentially — at no time will the hook script run in more than
one instance. However, other download workers may replace certificate files
while a hook is running.

### Shared hook (`-shared-hook`)

Called once after all per-domain hooks complete:

```
<script> shared <DOMAIN>...
```

Environment variable `DOMAINS` contains the space-separated list of all
domains.


## Password Notation

The `-passout` flag for PKCS12 protection accepts:

| Notation | Example | Source |
|----------|---------|--------|
| `pass:<password>` | `pass:secret123` | Literal value |
| `file:<path>` | `file:/etc/cert-proxy/p12pass` | Read from file |
| `env:<variable>` | `env:P12_PASSWORD` | Read from environment |


## Output Files

### PEM format (Linux default)

```
<certbase>/<domain>/cert.pem
<certbase>/<domain>/privkey.pem      (mode 0660)
<certbase>/<domain>/chain.pem
<certbase>/<domain>/fullchain.pem
```

Files are written with a timestamp infix (e.g., `cert-1714400000.pem`)
and then symlinked to the final name for atomic updates.

### PKCS12 format (Windows default)

```
<certbase>/<domain>/bundle.pfx       (mode 0660)
```


## Known Issues

- [#14](https://forgejo.schlittermann.de/heiko/cert-proxy/issues/14) — Hook: TIMESTAMP parameter not set
- [#13](https://forgejo.schlittermann.de/heiko/cert-proxy/issues/13) — PKCS12 on Windows with Java Keystore
- [#12](https://forgejo.schlittermann.de/heiko/cert-proxy/issues/12) — Client crashes if there are no certificates to fetch
- [#10](https://forgejo.schlittermann.de/heiko/cert-proxy/issues/10) — Shared hook should run only if certs are modified
- [#5](https://forgejo.schlittermann.de/heiko/cert-proxy/issues/5) — Remove certs no longer available on the server
