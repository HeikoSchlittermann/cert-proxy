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

* The cert-proxy is written in Go. If you need to build the binaries, you
  need to install a Go [build environment](https://golang.org).

* Next you need to setup the your local system to use a "go workspace":
  Create a `/etc/profile.d/local.sh` and add these two lines:

    export GOPATH=/usr/local/go
    PATH+=$GOPATH/bin

  Logout and login again.

* Next create the Go workspace:

    install -d $GOPATH

### Get the source, build, and install the binaries

* Clone the Git repository:

    cd /usr/local/go/src
    git clone http://git.schlittermann.de/user/heiko/cert-proxy

* To build the binaries, change your working directory into the project
  dir and build the binaries for your platforms:

    cd /usr/local/go/src/git-proxy
    make			    (1)
    GOOS=windows make		    (2)

  (1) Without the GOOS environment variable, the binaries for the current
  platform (probably "linux") are built.

  (2) With setting the *GOOS* environment variable, you can build the
  binaries for alternative platforms.

* The last step is installing the binaries for the current platform:

    make install

This should install the cert-proxy-server and cert-proxy-client into Go's bin
directory ($GOPATH/bin). If you want to install *only* the server or
*only* the client, use `make install-server` or `make install-client`

## Setup the cert-proxy-server

* You may want to use the supplied systemd-service files from `systemd/`.

    systemctl enable systemd/cert-proxy-server.service

* For mutual authentication X509 is used. To support you, a simple CA is
  part of the cert-proxy package. You may install this minimalistic CA:

    make install-ca

  This installs the CA into `/etc/cert-proxy/ca`. You may override the
  root directory by setting the *DESTDIR* environment variable.

* Once you have the CA, you can create the cert-proxy-ca

    cd /etc/cert-proxy/ca
    ./bin/mkssl-pem cert-proxy

* Copy the resulting file (`ca/cert-proxy.pem`) as
  `/etc/cert-proxy/ssl.pem`.

* Create the client-config-directory

    install -d /etc/cert-proxy/clients

* Start the service

    systemctl start cert-proxy-server

## Setup the Client

* Create a client authentication file

    cd /etc/cert-proxy/ca
    bin/mkssl-pem <client-name>

* Copy the <client-name>-ssl.pem as ssl.pem, and the cert-proxy-client(.exe) to the
  client system (the binaries are located in $GOPATH/bin/)

* Create the directory *certbase* for the downloaded clients

* Start the client

    cert-proxy-client [-sslfile ssl.pem] -certbase <certbase-dir> -verbose

  This will start the client, download the certs and re-download the
  certs once per tick interval (option -tick)

## Daily operation

Whenever a new cert should be made available for a client (or a number
of clients), add the new cert domain to the client-config file
`/etc/cert-proxy/clients/<client>`.

Add (line by line) the CNs for certificates the client is allowd to
access. Comments (#) are allowed

No restart required.

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

Yes, for PEM, multiple requests are used currently.
