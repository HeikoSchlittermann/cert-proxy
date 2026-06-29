<!-- (co)authored by ai:claude-opus-4-6 -->
# cert-proxy demo (podman)

Self-contained, two-container demo of cert-proxy distributing a **real
Let's Encrypt production certificate** for `elbtal.com`, obtained via the
DNS-01 challenge on `_ius.dns`.

```
_ius.dns (prod)                  local podman
┌────────────────────────┐   ┌──────────┐  mTLS  ┌──────────┐  shared vol  ┌──────────┐
│ dehydrated --cron      │scp│ cp-server│◄──────►│ cp-client│─────────────►│ cp-nginx │
│  DNS-01 via TSIG       │──►│  :4433   │ :4433  │ pulls    │ cpclient-    │ HTTPS    │
│  → LE prod cert        │   │ certbase/│        │ certs    │ certs vol    │ :8443    │
└────────────────────────┘   └──────────┘        └──────────┘              └──────────┘
```

## Layout

| Path                                    | What                                                          |
| --------------------------------------- | ------------------------------------------------------------- |
| `bin/`                                  | statically-built `cert-proxy-server` / `cert-proxy-client`    |
| `pki/`                                  | throwaway CA + server/client mutual-TLS certs (NOT the LE cert) |
| `server/certbase/elbtal.com/`           | the real LE cert pulled from `_ius.dns`                       |
| `server/clients/cp-client`              | authorizes `cp-client` for `elbtal.com`                       |
| `client/`                               | client mutual-TLS bundle + logging hook                       |
| `nginx/elbtal.conf`                     | vhost serving HTTPS with the delivered cert                   |
| `compose.yml`, `Containerfile`, `demo`  | orchestration                                                 |

The `cp-nginx` container mounts the client's `cpclient-certs` volume
read-only and serves `https://elbtal.com:8443/` using the delivered
`fullchain.pem` + `privkey.pem`. It waits for the cert to appear, then
starts; `./demo run` reloads it to pick up refreshed certs.

The `pki/` CA only authenticates client↔server. It is unrelated to
Let's Encrypt, which signs the actual `elbtal.com` certificate.

## Run

```shell
./demo up      # build image, start cp-server + cp-client
./demo run     # list → fetch → nginx HTTPS → 304 → authz walkthrough
./demo logs    # server log
./demo down    # tear down
```

## How the elbtal.com cert was obtained (on _ius.dns, prod)

```shell
acme elbtal.com                      # CNAME + TSIG key + domains.txt
dehydrated --cron --domain elbtal.com
# → /var/lib/dehydrated/certs/elbtal.com/{cert,privkey,chain,fullchain}.pem
```

Those four PEMs were copied into `server/certbase/elbtal.com/`.

## Notes

- Certs are real and expire 2026-09-26; `server/certbase` is a snapshot,
  not live-renewed.
- No cleanup was performed on `_ius.dns`: the CNAME, TSIG key and the
  `domains.txt` entry remain in place (so the demo can be refreshed).
- Static binaries → base image is irrelevant (here: `debian:trixie-slim`).
- Browser check: add `127.0.0.1 elbtal.com` to `/etc/hosts`, then open
  <https://elbtal.com:8443/> — it validates against the public LE chain.
