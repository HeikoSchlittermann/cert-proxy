# Plan: enterprise readiness

(co)authored by ai:claude-fable-5-1

Source: review of `e2d2a70` on 2026-09-26. Findings marked *verified* were
reproduced (server run against `demo/pki`, curl); the rest are from code
inspection. Effort figures are guesswork.

Not a finding: the tracked path `demo/server/dehydrated/private/_acme-challenge.elbtal.com`
is a symlink (mode 120000); no secret is in git history (all-blob scan done).

## Decisions needed before Phase 2

- [ ] Target customer: Windows endpoints (IIS/Exchange) vs Linux-only shops
      vs both. Drives Phase 2 item W and Phase 4 packaging order.
- [ ] Revocation model: CRL file vs short-lived client certs vs both.
- [ ] Support commitment to publish: supported versions, response time.
- [ ] Public home: stay on Forgejo, mirror to GitHub/Codeberg, or both.

---

## Phase 1 — Table stakes (~days)

### 1.1 CI that proves the build

No test/lint CI exists; `.forgejo/workflows/` only has the `nagonag`
dependency bot.

- [x] `.forgejo/workflows/ci.yaml` on push + PR: `go build ./...`,
      `go test ./...`, `golangci-lint run ./...`, `govulncheck ./...`,
      `go generate ./... && git diff --exit-code man/`, `gzip -t man/*.gz`.
      *Done;* the man page check compares decompressed content, as the
      compressed bytes differ across Go versions.
- [x] Scheduled job (weekly) running `make test-packaging`
      (needs podman on the runner — verify availability first).
      *Workflow in place; runner podman support still unverified.*
- [x] Fix current lint debt: `internal/program/program.go:21` (wsl_v5 ×2).

### 1.2 Server hardening

- [x] `cmd/cert-proxy-server/main.go:73` — replace `http.Serve(listener, nil)`
      with explicit `http.ServeMux` + `http.Server{ReadHeaderTimeout,
      ReadTimeout, WriteTimeout, IdleTimeout}`; graceful shutdown on
      SIGTERM/SIGINT.
- [x] `cmd/cert-proxy-server/serve.go` (`cert|chain|fullchain|privkey` branch
      and `bundle` branch) — *verified*: unauthenticated `/v1/cert/<unknown>`
      returns 500 with `open certs/<x>/cert.pem: no such file or directory`
      (leaks `-certbase`). Map `fs.ErrNotExist` → 404 with opaque body; other
      errors → 500 with opaque body, details only in the log.
- [x] `cmd/cert-proxy-server/version.go:14` — *verified*: `X-Version` sent to
      unauthenticated clients. Send only when `ctx[REMOTE] != ""`, or make it
      a flag.
- [x] `cmd/cert-proxy-server/main.go` TLS config — set
      `MinVersion: tls.VersionTLS12` explicitly (Go default today, *verified*
      1.0/1.1 rejected); add `-tls-min 1.2|1.3`.
- [x] `cmd/cert-proxy-server/pkcs12.go:70` — default encoder is
      `pkcs12.LegacyDES` when `pkcs12-compat` is absent. Make `modern` the
      default; `legacy` only on explicit request.
- [ ] `serve.go` bundle path — refuse empty PKCS12 password unless the
      client sent `pass=` explicitly (define semantics: empty `pass=` allowed,
      absent not).
      *Decided otherwise for now:* absent `pass` is served with an empty
      password and a deprecation warning in the server log, so v1.21
      clients keep working. Rejecting it is left for the next major
      release.
- [x] Follow-up: the startup check for `-certbase`/`-ccd` tests existence
      and type only, not readability (`cmd/cert-proxy-server/init.go`
      `checkDir`).
      *Done:* read and search access checked (`Readdirnames(1)`); write
      access deliberately not required, the server never writes.

### 1.3 Client hardening

- [x] `cmd/cert-proxy-client/cert/const_windows.go` — `PKCS12Compat = "legacy"`
      → `"modern"`. Document the Windows import caveat instead.
- [x] `cmd/cert-proxy-client/init.go` `checkConnectURL` — reject `http://`
      (private keys over plaintext); keep `https` only, or add
      `-insecure-http` with a loud warning.
      *Done:* https only, no `-insecure-http`.
- [x] `cmd/cert-proxy-client/cert/cert.go` `Execute` and
      `cmd/cert-proxy-client/main.go` `fetchCNs` — no timeouts anywhere
      (`grep Timeout` → 0 hits). Add `-timeout` (default e.g. 60s) applied
      via `http.Client{Timeout}` or per-request `context.WithTimeout`.
- [x] `cmd/cert-proxy-client/secret/secret.go:17,36` — index panic on
      `-passout foo` (no colon) and explicit `panic` on unknown scheme.
      Return errors.
- [x] `cmd/cert-proxy-client/init.go:89` — help text typo `-servernae`.

### 1.4 systemd units

- [ ] `systemd/cert-proxy-server.service` — currently root with only
      `ProtectSystem=strict`. Add `User=`/`Group=` (sysusers entry),
      `NoNewPrivileges=yes`, `PrivateTmp=yes`, `ProtectHome=yes`,
      `ProtectKernelTunables=yes`, `ProtectControlGroups=yes`,
      `RestrictAddressFamilies=AF_INET AF_INET6`,
      `CapabilityBoundingSet=` (+ `AmbientCapabilities=CAP_NET_BIND_SERVICE`
      only if port <1024 is supported). `[Install] WantedBy=multi-user.target`,
      `[Unit] After=network-online.target Wants=network-online.target`.
      Document how the service user gets read access to the ACME cert store
      (group or ACL).
      *Partly, decided otherwise:* sandboxing done, but the unit stays
      root (upgrade safety: `server-ssl.pem` and dehydrated's store are
      root-only and renewals re-create them so) with
      `CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_DAC_READ_SEARCH`.
      `DynamicUser=yes` + `SupplementaryGroups=ssl-cert` is an opt-in
      drop-in (`systemd/cert-proxy-server.dynamic-user.conf`). Default
      flip is a candidate for the next major release.
- [x] `systemd/cert-proxy-client.service` — same hardening set; the client
      needs write to `-certbase` and exec of the hook, so `ProtectSystem=full`
      + `ReadWritePaths=/var/lib/cert-proxy`.
- [ ] `systemd/cert-proxy-server.service` `-certbase /var/lib/dehydrated/certs`
      — dehydrated-specific. Move into `/etc/default/cert-proxy-server` as a
      documented example alongside certbot (`/etc/letsencrypt/live`) and
      acme.sh layouts.
      *Decided otherwise:* stays in `ExecStart` before `$OPTS` (last
      flag wins) so a locally modified conffile cannot lose it on
      upgrade; dehydrated and certbot examples are in the `.default`
      file. No acme.sh example yet.
- [ ] Follow-up: `test/packaging` does not assert that
      `usr/share/doc/cert-proxy-server/examples/dynamic-user.conf` and
      the server's sysusers snippet land in the server package.
- [x] Update `man/cert-proxy-server.8.md`, `man/cert-proxy-client.8.md`,
      `.gogogo.conf` (sysusers) and regenerate man pages.

### 1.5 Scrub author-specific infrastructure from shipped files

- [x] `systemd/cert-proxy-client.default:4` — `cert-proxy.schlittermann.de`
      → `cert-proxy.example.com`.
- [x] `CA/lib/vars.sh.example` — `DE/Sachsen/Dresden/IUS` → neutral
      placeholders with a comment.
- [x] `README.md:45` clone URL; `README.md:250,338-342` issue links → plain
      `#N` references plus one "issue tracker" link in a Support section.
- [x] `.gogogo.conf:268` `publish.deb: dupload` — private repo. Either move
      to a non-tracked override or document that operators must set their
      own destination.

Gate for Phase 1: CI green; `go test ./...`; manual curl checks for 404/opaque
body; `systemd-analyze security cert-proxy-server.service` reported.
Status: `go test ./...`, curl checks (`bin/verify-phase1`) and
`systemd-analyze security` (4.5 as shipped, 3.8 with the drop-in) done;
"CI green" open until the workflow has run on the runner.

---

## Phase 2 — Security model (~1–2 weeks)

### 2.1 Revocation (decision above)

- [ ] Option A: `-crl <file>`; verify in `tls.Config.VerifyPeerCertificate`;
      reload on SIGHUP and on mtime change. Provide `CA/bin/mkcrl` (or the Go
      CA below).
- [ ] Option B: `CA/lib/openssl.cnf:73` `default_days 3650` → ≤365; document
      renewal via `mkssl-pem`.
- [ ] Add explicit disable marker for a client (e.g. empty file or
      `clients/<cn>.disabled`) with a distinct log line; document in
      `man/cert-proxy-clients.5.md`.

### 2.2 Audit log and observability

- [ ] One structured line per `privkey`/`bundle` delivery: timestamp, CN,
      domain, remote addr, result, bytes. Independent of `-verbose`.
- [ ] `-log-format text|json`.
- [ ] `/healthz` (unauthenticated, no data) on the main listener; optional
      `-metrics <addr>` Prometheus endpoint (request counts by endpoint/status,
      TLS handshake failures). Separate listener so it can be firewalled.
- [ ] Document journald/syslog integration and what to alert on.

### 2.3 Windows client (decision above)

- [ ] Either set owner-only DACL on written `.pfx`/`.pem` (needs
      `golang.org/x/sys/windows` — new dependency; weigh against
      `CGO_ENABLED=0` cross-build, it is pure Go so OK) or document that the
      target directory ACL is the only protection.
- [ ] Windows guide: scheduled task / service wrapper, `-certbase` location,
      hook example in PowerShell importing the `.pfx` into the machine store.

### 2.4 CA tooling

- [ ] Minimum: `CA/lib/mkca` `genrsa -des3` → `-aes256`; document offline
      storage of `private/cakey.pem`; `mkcrl` script; drop `git`-in-CA-dir as
      a hard requirement (make optional).
- [ ] Better (tracked as README #18): `cert-proxy-server ca init|issue|revoke|crl`
      in Go, removing the openssl CLI and bash dependency and enabling the
      Windows side to run a CA too.

Gate for Phase 2: a client cert revoked via the chosen mechanism is refused
(*test in `cmd/cert-proxy-server/server_test.go`*); audit line present in
`integration_test.go`; `/healthz` covered by a test.

---

## Phase 3 — Product and governance (~1 week, mostly writing)

- [ ] `SECURITY.md`: threat model (what a stolen client cert yields; what the
      server trusts; why `cert/chain/fullchain` are public; PKCS12 password
      travels in the query string over TLS only), disclosure contact,
      supported versions.
- [ ] `CHANGELOG.md` (Keep-a-Changelog) + semver policy statement; backfill
      from tags at least for the last two minors.
- [ ] `docs/operations.md` runbook: install, first CA, onboard a client,
      rotate server cert, rotate client cert, revoke, backup (CA key,
      `clients/`, server-ssl.pem), restore, upgrade, HA statement (stateless
      behind a TCP LB when certbase is shared/replicated; or "single
      instance" explicitly).
- [ ] `CONTRIBUTING.md` (build, lint, man page regen, DCO or CLA decision).
- [ ] Demo runnable by outsiders: replace the live DNS-01 flow with `pebble`
      (Let's Encrypt test CA) or a shipped self-signed "fake LE" chain; remove
      `_ius.dns`, `elbtal.com`, TSIG references from `demo/README.md`,
      `demo/compose.yml`, `demo/server/dehydrated/*`.
- [ ] Support section in README: tracker URL, security contact, versioning.

---

## Phase 4 — Distribution (open-ended, depends on market decision)

- [ ] RPM: enable `artifacts.rpm` in `.gogogo.conf`; test install in a
      Rocky/Alma container (extend `test/packaging/`).
- [ ] Container image (distroless/static, both binaries) + Compose example;
      Helm chart only if asked.
- [ ] Release artifacts: document `.sig` verification; attach SBOM
      (`cyclonedx-gomod` or `syft`).
- [ ] Publish destination for outsiders (Forgejo packages or a public apt
      repo) instead of `dupload`.
- [ ] Mirror repository to GitHub/Codeberg (module path can stay).

---

## Out of scope / explicitly not planned

- Rewriting the protocol (v1 stays; any change is `/v2`).
- Removing the "public cert/chain endpoints" design; documented instead.
