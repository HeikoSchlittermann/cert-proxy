% cert-proxy 7 "August 2026" "cert-proxy" "Miscellaneous Information"

# NAME

cert-proxy - certificate distribution protocol and on-disk layout

# DESCRIPTION

Cert-proxy distributes existing TLS certificates from one host to many. A
central **cert-proxy-server**(8) publishes a tree of certificates over HTTPS;
each **cert-proxy-client**(8) authenticates with an X509 client certificate and
downloads the material it is authorized for.

Cert-proxy neither obtains nor renews certificates. Something else, typically
**dehydrated**(1) driving an ACME certificate authority, maintains the tree the
server publishes.

# TRUST MODEL

Both sides authenticate with certificates issued by a local CA that exists only
for cert-proxy; the scripts under */etc/cert-proxy/ca* create and operate it.

The client verifies the server certificate against that CA and checks the host
name (or IP address) in its Subject Alternative Name extension. The expected
name defaults to the host the client connects to and can be overridden with
**-servername**. The server requires a client certificate for everything
except the public endpoints below, and uses the common name of that certificate
as the client's identity. Authorization is then a per-client list of domains, one
file per common name, described in **cert-proxy-clients**(5).

Note that the client certificate's common name is an identity, not a domain
claim: a client whose common name is *www.example.com* is authorized for exactly
the domains listed in its own file, which need not include *www.example.com*.

# ENDPOINTS

All requests have the form **/v1/***req*[**/***domain*]. A path with more
segments than that is answered with **400 Bad Request**, and one that matches no
endpoint at all with **404 Not Found**.

*/v1/list*
: Requires a client certificate. Returns the domains the requesting client is authorized for, one per line, taken from that client's own authorization file. It is not a catalogue of everything the server holds.

*/v1/cert/DOMAIN*
: Public. The certificate.

*/v1/chain/DOMAIN*
: Public. The CA chain.

*/v1/fullchain/DOMAIN*
: Public. Certificate and chain.

*/v1/privkey/DOMAIN*
: Requires a client certificate authorized for *DOMAIN*. The private key.

*/v1/bundle/DOMAIN*
: Requires a client certificate authorized for *DOMAIN*. A PKCS12 bundle. If *bundle.p12* exists beside the other files it is served as is; otherwise it is generated on the fly from the stored certificate and key.

That the first three endpoints are public is deliberate: a certificate and its
chain are published to every TLS client anyway. Only the private key and the
bundle that contains it are restricted.

# QUERY PARAMETERS

**format**=*PEM*|*PKCS12*
: Selects the *.pem* or *.p12* file extension within the domain directory. **PFX** and **P12** are accepted as synonyms of **PKCS12**. Case-insensitive, default **PEM**. Any other value gives **400 Bad Request**.

**pkcs12-compat**=*legacy*|*modern*
: Algorithm profile used when a bundle is generated on the fly. **legacy** is for consumers that cannot read modern PKCS12 encryption, such as older Windows and Java releases.

**pass**=*password*
: Password protecting a bundle that is generated on the fly.

# RESPONSES

Responses that reach the content handler carry an **x-version** header naming
the server's version. Responses rejected earlier, by authentication or
authorization, do not.

Failures on the authenticated endpoints are all **401 Unauthorized**, but the
body differs by cause: *no (valid) client certificate* when none was presented,
*unauthorized* when the client has no authorization file or its common name is
unusable as a file name, and *client CN not authorized for DOMAIN* when the file
exists but does not list the domain. The last one echoes the requested domain and
the client's own common name back to the peer.

Content is served with **If-Modified-Since** support, so a client that already
holds a current copy receives **304 Not Modified**. This is what makes a frequent
timer cheap.

A request for a domain the server does not hold yields **500 Internal Server
Error** rather than **404 Not Found**, because the missing file is reported as an
ordinary read failure.

# FILE LAYOUT

On the server, below the directory named by **-certbase**, each domain is a
directory:

```
certs/example.com/cert.pem
certs/example.com/chain.pem
certs/example.com/fullchain.pem
certs/example.com/privkey.pem
```

An optional *bundle.p12* is served in preference to generating one.

On the client, below the directory named by **-certbase**, the same names appear
per domain for PEM; a PKCS12 bundle is stored as *bundle.pfx*, because Windows
dislikes *.p12* there. On Unix each name is a symlink to a file carrying the timestamp of
the version it holds, so a replacement is atomic and the previous version stays
on disk; **-symlink=false** writes the plain file instead. On Windows the plain
file is replaced by rename. Private keys are written with mode 0600, everything
else with 0644.

# SEE ALSO

**cert-proxy-client**(8), **cert-proxy-server**(8), **cert-proxy-clients**(5),
**dehydrated**(1)

# AUTHORS

Heiko Schlittermann <hs@schlittermann.de>
