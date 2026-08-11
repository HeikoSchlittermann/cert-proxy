% cert-proxy-client 8 "August 2026" "cert-proxy" "System Administration"

# NAME

cert-proxy-client - fetch certificates from a cert-proxy server

# SYNOPSIS

**cert-proxy-client** [*options*] [*CN*]...

**cert-proxy-client** **man** [*section*] [*page*]

**cert-proxy-client** **-help** | **-version**

# DESCRIPTION

**cert-proxy-client** is the client side of the cert-proxy suite. It connects to
a **cert-proxy-server**(8) over HTTPS, authenticates with an X509 client
certificate, and downloads the certificate material for one or more domains.

Normally it is not run by hand but by the supplied
*cert-proxy-client.timer* systemd unit.

Domains are selected in one of three ways, which may be combined: the *CN*
positional arguments, a list file given with **-cnfile**, and - unless
**-auto** is disabled - the list the server itself offers via its
**/v1/list** endpoint. If **-auto** is disabled and neither **-cnfile** nor a
positional *CN* is given, the client prints its usage and exits with status 1.

Each domain is dispatched to a worker pool. A worker downloads the artifacts
with an *If-Modified-Since* condition, so unchanged material is not transferred
again unless **-force** is given. Files are written atomically. On Unix the
current version of each file is exposed through a symlink pointing at a
timestamped file, unless **-symlink** is disabled; on Windows the file is
replaced by rename.

See **cert-proxy**(7) for the protocol and the on-disk layout.

# COMMANDS

**man** [*section*] [*page*]
: Display a manual page shipped inside the binary. With no argument the page **cert-proxy-client**(8) is selected. A single numeric argument selects a section, a single non-numeric argument is looked up as a page name in every section, and two arguments select section and page explicitly. When standard output is a terminal the page is rendered through **man**(1); otherwise the raw roff source is written to standard output.

# OPTIONS

**-auto**
: Fetch all CNs the server offers through its **/v1/list** endpoint. Default **true**. Use **-auto=false** to restrict the run to the domains named explicitly.

**-certbase** *dir*
: Base directory for the downloaded certificates. Default *certs*. The per-domain subdirectory is created on demand, but *dir* itself must already exist.

**-cnfile** *file*
: Read the domain list from *file*, or from standard input when *file* is **-**. The format is the one described in **cert-proxy-clients**(5).

**-connect** *[scheme://]server[:port]*
: Address of the cert-proxy server. Default *https://localhost:4433*. A missing scheme is taken as **https**, which implies port 443 unless a port is given. Trailing slashes are removed.

**-force**
: Download unconditionally, ignoring *If-Modified-Since*. Default **false**.

**-format** *PEM*|*PKCS12*
: Format of the requested certificates. **PEM** writes *cert.pem*, *chain.pem*, *fullchain.pem* and *privkey.pem*; **PKCS12** writes a single bundle. Default **PEM** on Unix and **PKCS12** on Windows. The value is case-insensitive.

**-hook** *file*
: Program to run for each domain once its certificates are done, whether they were fetched or found unmodified. See **THE HOOK SCRIPT** below.

**-jobs** *number*
: Maximum number of domains processed in parallel. Default: the number of CPUs.

**-passout** *scheme*:*password*
: Password protecting the PKCS12 bundle. Default: none. See **PASSWORD SOURCES** below.

**-pkcs12-compat** *legacy*|*modern*
: PKCS12 compatibility level requested from the server. Default **modern** on Unix and **legacy** on Windows.

**-servername** *CN*
: Common name expected in the server's certificate. When empty, the FQDN of the host being connected to is used. Default: empty.

**-shared-hook** *file*
: Program to run once after all per-domain hooks have finished. See **THE HOOK SCRIPT** below.

**-sslfile** *file*
: PEM file holding the client credentials: certificate, private key and CA. Default *client-ssl.pem*.

**-stderr** *stderr*|*stdout*
: Channel diagnostics are written to. Default **stderr** on Unix and **stdout** on Windows. Selecting **stdout** helps when running under PowerShell.

**-symlink**
: Expose the current file of each artifact through a symlink. Default **true** on Unix, **false** on Windows.

**-verbose**
: Report progress. Default **false**.

**-help**
: Print the usage to standard output and exit with status 0.

**-version**
: Print the program version and exit with status 0.

# ARGUMENTS

*CN*
: A domain to fetch. May be repeated. Each value is validated as a domain name; an invalid value is fatal.

# ENVIRONMENT

**JOURNAL_STREAM**
: Set by systemd. When present, timestamps are omitted from log lines because the journal records them already.

# FILES

*/etc/cert-proxy/client-ssl.pem*
: Client credentials used by the supplied systemd unit.

*/etc/cert-proxy/hook*
: Hook program shipped as a template by the Debian package. It is not executable as installed.

*/var/lib/cert-proxy/certs*
: Certificate store used by the supplied systemd unit. The Debian package creates it with mode 0750, owned by *root* and group *ssl-cert*.

*/etc/default/cert-proxy-client*
: Read by the systemd unit. The variable **OPTS** is appended to the command line.

# EXIT STATUS

**cert-proxy-client** exits with 0 on success and with a non-zero status if any
part of the run failed.

# THE HOOK SCRIPT

The program named by **-hook** is called for each domain as soon as that domain
is done. For **-format PEM** the call is:

```
hook deploy_cert DOMAIN KEYFILE CERTFILE FULLCHAIN CHAINFILE TIMESTAMP
```

For **-format PKCS12** it is:

```
hook deploy_cert DOMAIN BUNDLEFILE TIMESTAMP
```

The program named by **-shared-hook** is called once after all per-domain hooks
have finished:

```
shared-hook shared DOMAIN...
```

Environment variables of the same names as the positional parameters are set for
the child, possibly overriding variables inherited from the caller. The shared
hook additionally receives **DOMAINS**, a space separated list of the domains.

Hooks never run concurrently with each other: at no point is more than one hook
instance active. While a hook runs, however, other workers continue and may
replace certificate files the hook indirectly relies on.

On Windows, running a PowerShell hook may require
**set-executionpolicy remotesigned**.

# PASSWORD SOURCES

The value of **-passout** is a scheme followed by a colon:

**pass:***password*
: The password literally, on the command line.

**file:***path*
: The first line of *path*.

**env:***name*
: The value of the environment variable *name*.

# EXAMPLES

Fetch everything the server offers, with explicit credentials:

```
cert-proxy-client -connect https://cert-proxy.example.com \
    -servername cert-proxy.example.com \
    -sslfile client-ssl.pem \
    -verbose
```

Fetch two named domains only, and run a hook for each:

```
cert-proxy-client -auto=false \
    -connect https://cert-proxy.example.com \
    -hook /etc/cert-proxy/hook \
    www.example.com mail.example.com
```

The command line used by the supplied systemd unit is:

```
cert-proxy-client -verbose \
    -sslfile /etc/cert-proxy/client-ssl.pem \
    -certbase /var/lib/cert-proxy/certs $OPTS
```

# SEE ALSO

**cert-proxy**(7), **cert-proxy-clients**(5), **cert-proxy-server**(8),
**systemd.timer**(5)

# AUTHORS

Heiko Schlittermann <hs@schlittermann.de>
