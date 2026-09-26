% cert-proxy-server 8 "August 2026" "cert-proxy" "System Administration"

# NAME

cert-proxy-server - serve certificates to authenticated cert-proxy clients

# SYNOPSIS

**cert-proxy-server** [*options*]

**cert-proxy-server** **man** [*section*] [*page*]

**cert-proxy-server** **-help** | **-version**

# DESCRIPTION

**cert-proxy-server** is the server side of the cert-proxy suite. It serves the
certificate material below **-certbase** over HTTPS and requires an X509 client
certificate for every request that is not public.

The public part of a certificate is handed out without authentication. The
domain list requires a valid client certificate. Private keys and PKCS12 bundles
additionally require the client's common name to be authorized for the requested
domain, which is configured per client in the directory named by **-ccd** and
described in **cert-proxy-clients**(5).

The process runs in the foreground and is normally started by the supplied
*cert-proxy-server.service* systemd unit. It does not fork, write a pid file,
or reload its configuration on a signal; the authorization files are read on
each request, so changes there take effect immediately. On **SIGTERM** or
**SIGINT** it stops accepting connections and waits up to 10 seconds for
running requests to finish before it exits.

See **cert-proxy**(7) for the endpoints and the on-disk layout.

# COMMANDS

**man** [*section*] [*page*]
: Display a manual page shipped inside the binary. With no argument the page **cert-proxy-server**(8) is selected. A single numeric argument selects a section, a single non-numeric argument is looked up as a page name in every section, and two arguments select section and page explicitly. When standard output is a terminal the page is rendered through **man**(1); otherwise the raw roff source is written to standard output.

# OPTIONS

**-ccd** *dir*
: Directory holding the per-client authorization files. Default *clients*. A file is looked up by the client certificate's common name. See **cert-proxy-clients**(5). Must be an existing directory, see below.

**-certbase** *dir*
: Base directory the certificates are served from. Default *certs*. Each domain is a subdirectory holding *cert.pem*, *chain.pem*, *fullchain.pem* and *privkey.pem*. Must be an existing directory, see below.

**-idle-timeout** *duration*
: Maximum time to wait for the next request on a keep-alive connection. Default *2m0s*. With **0** the value of **-read-timeout** is used.

**-read-header-timeout** *duration*
: Maximum time for reading the request headers, starting when the connection is accepted, which includes the TLS handshake. Default *10s*. **0** disables the limit.

**-read-timeout** *duration*
: Maximum time for reading the entire request. Default *30s*. **0** disables the limit.

**-serve** *[host]:port*
: Address the TLS listener binds to. Default *:4433*.

**-sslfile** *file*
: PEM file holding the server credentials: certificate, private key and CA. Default *server-ssl.pem*. Clients that are started without **-servername** expect the certificate's Subject Alternative Name extension to contain the host name or IP address they connect to.

**-tls-min** *1.2*|*1.3*
: Minimum TLS protocol version accepted. Default *1.2*. Any other value is fatal.

**-verbose**
: Report every request. Default **false**.

**-write-timeout** *duration*
: Maximum time from the end of reading the request headers until the response is written. Default *1m0s*. **0** disables the limit.

Durations use Go duration syntax, for example *10s*, *2m* or *1h30m*.

If an option is given more than once, the last occurrence wins.

Before the listener starts, **cert-proxy-server** refuses to run if **-certbase**
or **-ccd** does not name an existing, readable directory (or a symlink to
one), and says which of the two it is and why: missing, not a directory, a
symlink pointing nowhere, or not readable. A wrong path would otherwise only
surface as 404 or 401 answers. Write access is never required: the server does
not write to the file system. A readable directory does not guarantee readable
files below it; an unreadable certificate or key still fails the individual
request.

**-help**
: Print the usage to standard output and exit with status 0.

**-version**
: Print the program version and exit with status 0.

# ENVIRONMENT

**INVOCATION_ID**
: Set by systemd. When present, timestamps are omitted from log lines because the journal records them already.

# FILES

*/etc/cert-proxy/server-ssl.pem*
: Server credentials used by the supplied systemd unit.

*/etc/cert-proxy/clients/*
: Per-client authorization files used by the supplied systemd unit.

*/var/lib/dehydrated/certs*
: Certificate store passed as **-certbase** by the supplied systemd unit. Cert-proxy does not create or renew certificates; something else, for example **dehydrated**(1), maintains this tree. For certbot use */etc/letsencrypt/live*; see the comments in */etc/default/cert-proxy-server*.

*/etc/default/cert-proxy-server*
: Read by the systemd unit. The variable **OPTS** is appended to the command line, after the unit's own options, so an option in **OPTS** overrides the unit's: for example **OPTS=**"*-certbase /etc/letsencrypt/live*" replaces the unit's default **-certbase** */var/lib/dehydrated/certs*. As shipped, **OPTS** is not set.

*/usr/share/doc/cert-proxy-server/examples/dynamic-user.conf*
: Example drop-in to run the service unprivileged, see below.

*/etc/cert-proxy/ca/*
: Helper scripts that create and operate the local CA issuing the client certificates, installed by the Debian package or by **make install-ca**.

The supplied *cert-proxy-server.service* runs as root, with the capability
bounding set reduced to **CAP_NET_BIND_SERVICE** (ports below 1024, for example
**-serve** *:443*) and **CAP_DAC_READ_SEARCH** (reading certificate files owned
by another user, for example a dedicated dehydrated user), plus the usual
systemd sandboxing (**ProtectSystem=strict**, **ProtectHome=yes** and others):
**-certbase**, **-ccd** and **-sslfile** must not point below */home*, */root*
or */run/user*.

To run it as an unprivileged dynamic user instead (**DynamicUser=yes** with the
supplementary group *ssl-cert*, which the Debian package creates through a
*sysusers.d* snippet, and no capabilities at all), install the shipped example
drop-in:

```
install -D -m 0644 /usr/share/doc/cert-proxy-server/examples/dynamic-user.conf \
    /etc/systemd/system/cert-proxy-server.service.d/dynamic-user.conf
systemctl daemon-reload && systemctl restart cert-proxy-server
```

Before that, *server-ssl.pem*, the *clients/* directory with its files, and the
ACME certificate store (including every per-domain directory and *privkey.pem*)
must be readable by group *ssl-cert* (directories 0750, files 0640), for
example:

```
chgrp ssl-cert /etc/cert-proxy/server-ssl.pem
chmod 0640 /etc/cert-proxy/server-ssl.pem
chgrp -R ssl-cert /etc/cert-proxy/clients /var/lib/dehydrated/certs
chmod -R g+rX /etc/cert-proxy/clients /var/lib/dehydrated/certs
```

This must survive renewals: ACME clients re-create the files on every renewal,
**dehydrated**(1) with mode 0600. Repeat the **chgrp**/**chmod** from a
dehydrated *deploy_cert* hook or a certbot **--deploy-hook**. Under the drop-in
ports below 1024 are not supported; keep the default port 4433.

# EXIT STATUS

**cert-proxy-server** exits with 0 when asked for **-help** or **-version**, and
with a non-zero status when it cannot start, for example because the credentials
are missing or the listener address is in use.

# EXAMPLES

Serve a dehydrated certificate tree on all interfaces:

```
cert-proxy-server -verbose \
    -sslfile /etc/cert-proxy/server-ssl.pem \
    -certbase /var/lib/dehydrated/certs \
    -ccd /etc/cert-proxy/clients
```

Authorize the client whose certificate common name is *www.example.com* for two
domains:

```
printf '%s\n' example.com www.example.com \
    > /etc/cert-proxy/clients/www.example.com
```

# SEE ALSO

**cert-proxy**(7), **cert-proxy-clients**(5), **cert-proxy-client**(8),
**systemd.service**(5)

# AUTHORS

Heiko Schlittermann <hs@schlittermann.de>
