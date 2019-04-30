# CERT-PROXY

cert-proxy should act as a proxy between systems with limited access to
the Let's Encrypt CA or with limited abilities to respond to the
challenges.

## Concept

Certproxy receives cert requests from clients and provides the result to
the clients.


      { LE CA }
         | |
         ^ |
	 | v
	 | |
       LE helper (dehydrated)---> DNS
       cert-proxy-server
         | |
	 ^ v
	 | |
       cert-proxy-client

As the cert-proxy-server may send sensitive information to the client, the
client has to authenticate itself. As the client has to trust the
information from the proxy, the server needs to authenticate with the
client.

## Installation

### Prepare the build environment

The cert-proxy is written in Go. If you need to build the binaries, you
need to install a Go [build environment](https://golang.org).

Next you need to setup the your local system to use a "go workspace":
Create a `/etc/profile.d/local.sh` and add these two lines:

    export GOPATH=/usr/local/go
    PATH+=$GOPATH/bin

Logout and login again.
Next create the Go workspace:

    install -d $GOPATH

### Get the source and build and install the binaries

Clone the Git repository:

    cd /usr/local/go/src
    git clone http://git.schlittermann.de/user/heiko/cert-proxy

To build the binaries, change your working directory into the project
dir and build the binaries for your platforms:

    cd /usr/local/go/src/git-proxy
    make			    (1)
    GOOS=windows make		    (2)

(1) Without the GOOS environment variable, the binaries for the current
platform (probably "linux") are built.

(2) With setting the *GOOS* environment variable, you can build the
binaries for alternative platforms.

The last step is installing the binaries for the current platform:

    make install

This should install the cert-proxy-server and cert-proxy-client into Go's bin
directory ($GOPATH/bin). If you want to install *only* the server or
*only* the client, use `make install-server` or `make install-client`

### Setup the CA

For mutual authentication X509 is used. To support you, a simple CA is
part of the cert-proxy package. You may install this minimalistic CA:

    make install-ca

This installs the CA into `/etc/cert-proxy/ca`. You may override the
root directory by setting the *DESTDIR* environment variable.

## Updates

Once the cert-proxy sub-packages (server, client, ca) are installed, you
may regularly check for updates:

    cd /usr/local/go/src/cert-proxy
    git pull

and repeat the installation steps.


## Operation

### Initial setup of the CA

If you run your own CA, you may skip this step.

For authentication between client and proxy X509 is used (in both
directions).  Client and Proxy use a single file containing the
information they need (default name: ssl.pem). You're encouraged to use
the provided minimalistic CA.

First create the CA:

    cd /etc/cert-proxy/ca
    cp lib/vars.sh.example lib/vars.sh
    <edit> lib/vars.sh
    ./lib/mkca

Once the CA is created, create a ssl bundle for the cert proxy:

    ./bin/mkssl-pem <proxy-host-name>

The proxy's hostname does not matter, as you can override the expected
name on the client. The client's hostname is used as an authorization
key to access the certificates. Copy the resulting files to a safe place
for later use by the cert proxy server.

    cert-proxy -serve :443

### Setup a client

Login to the server and make use of it's CA:

    cd CA
    ./bin/mkssl-pem <proxy-client-name>

The proxy-client-name doesn't matter as long as it is uniq among all
clients. Copy the resulting -ssl.pem file to your client. (WARNING: It
contains private *unprotected* key material)

Create a client access configuration on the server:

    cd <working-dir-of the server>
    mkdir clients
    <edit> clients/<proxy-client-name>

Add (line by line) the CNs for certificates the client is allowd to
access. Comments (#) are allowed

Login to the client. Install the cert-proxy-client binary, and start it.

    cert-proxy-client -connect https://proxy example.com


## The API

Authentication in both directions is done with X509 certs, issued by a
CA running on the cert-proxy.

Auhentication (authn) is always required. Authorization (authz) for
specific commands

    Requests:
    GET /v1/list			// no authz
    GET /v1/cert/<domain>		// cert, no authz¹
    GET /v1/chain/<domain>		// chain, no authz¹
    GET /v1/fullchain/<domain>		// fullchain, no authz¹
    GET /v1/privkey/<domain>		// privkey, authz required¹
    GET /v1/bundle/<domain>		// all of the above, authz required¹

¹) a ?format=[pem|pkcs12] may be appended. The default format is "pem"

Yes, for PEM, multiple requests are used.
