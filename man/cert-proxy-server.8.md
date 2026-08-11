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
each request, so changes there take effect immediately.

See **cert-proxy**(7) for the endpoints and the on-disk layout.

# COMMANDS

**man** [*section*] [*page*]
: Display a manual page shipped inside the binary. With no argument the page **cert-proxy-server**(8) is selected. A single numeric argument selects a section, a single non-numeric argument is looked up as a page name in every section, and two arguments select section and page explicitly. When standard output is a terminal the page is rendered through **man**(1); otherwise the raw roff source is written to standard output.

# OPTIONS

**-ccd** *dir*
: Directory holding the per-client authorization files. Default *clients*. A file is looked up by the client certificate's common name. See **cert-proxy-clients**(5).

**-certbase** *dir*
: Base directory the certificates are served from. Default *certs*. Each domain is a subdirectory holding *cert.pem*, *chain.pem*, *fullchain.pem* and *privkey.pem*.

**-serve** *[host]:port*
: Address the TLS listener binds to. Default *:4433*.

**-sslfile** *file*
: PEM file holding the server credentials: certificate, private key and CA. Default *server-ssl.pem*. Clients that are started without **-servername** expect the certificate's common name to be the FQDN they connect to.

**-verbose**
: Report every request. Default **false**.

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
: Certificate store used by the supplied systemd unit. Cert-proxy does not create or renew certificates; something else, for example **dehydrated**(1), maintains this tree.

*/etc/default/cert-proxy-server*
: Read by the systemd unit. The variable **OPTS** is appended to the command line.

*/etc/cert-proxy/ca/*
: Helper scripts that create and operate the local CA issuing the client certificates, installed by the Debian package or by **make install-ca**.

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
