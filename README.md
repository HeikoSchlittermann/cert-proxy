# CERT-PROXY

Cert-proxy should act as a proxy between systems with limited access to
the Let's Encrypt CA or with limited abilities to respond to the
challenges.

## Concept

Cert-proxy receives cert requests from clients and provides the result to
the clients.

```
      [ Let's Encrypt CA ]
         | |
         ^ |
	       | v
      	 | |
       LE helper (dehydrated)---> DNS
       Cert-proxy server
         | |
         ^ v
         | |
       Cert-proxy client
```

As the Cert-proxy server may send sensitive information to the client, the
client has to authenticate itself. As the client has to trust the
information from the proxy, the server needs to authenticate with the
client.


## Installation

Currently there a packaged version for Debian systems only (see the
other branches of this repo).  For other platforms you need to build it
from the sources. You can build the binaries for both Linux and Windows
using the same build environment by using cross compilation. It is easy.
Read on.


### Prepare the build environment on a Linux system

* You need to install a Go [build environment](https://golang.org).
  I used Go 1.14.


### Get the source, build, and install the binaries

The source is published in a GIT
[repository](https://gitea.schlittermann.de/heiko/cert-proxy)

* Clone the Git repository:

  ```shell
  git clone http://gitea.schlittermann.de/heiko/cert-proxy
  ```

* To build and install the binaries, change your working
  directory into the project directory and build the binaries for your
  platforms:

  ```shell
  cd cert-proxy
  make install
  ```

  This should install the cert-proxy-server and cert-proxy-client into
  `/usr/local/bin/`. If you want to install *only* the server or
  *only* the client, use `make install-server` or `make install-client`

* Now build the binaries for the alternate platforms (e.g. windows)

  ```shell
  GOOS=windows make install
  ```

  This leaves the executables `/usr/local/bin/windows_amd64/`.


## Setup the cert-proxy-server

For the initial setup you need to do several steps. Setup the CA,
setup, setup the Cert-proxy-server and Systemd startup scripts and start the service.


### Setup the CA

* For mutual authentication between the client and the server X509 is
  used. To support you, a simple CA is part of the cert-proxy package.
  You may install this minimalistic CA:

  ```shell
  make install-ca
  ```

  This installs the CA into `/etc/cert-proxy/ca`.

* Configure the CA:

  ```shell
  cd /etc/cert-proxy/ca
  cp lib/vars.sh.example lib/vars.sh
  <edit> lib/vars.sh
  ```

* Initialize the CA:

  ```shell
  lib/mkca
  ```

* Now you have your own CA and you can start creating certificate
  "bundles". First you need a certificate for your cert-proxy-server.
  The name of the certificate doesn't matter, but the client expects it
  to be _cert-proxy_ per default.

  ```shell
  cd /etc/cert-proxy/ca
  ./bin/mkssl-pem cert-proxy.example.com cert-proxy
  ```

* Copy the resulting file (`cert-proxy-ssl.pem`) to the place where the
  Cert-proxy server (when started by Systemd) expects it.

  ```shell
  cp cert-proxy-ssl.pem /etc/cert-proxy/server-ssl.pem
  ```


### Create the config directory

The Cert-proxy server expects a directory with per-client configuration
files. This directory needs to exist:

```shell
install -d /etc/cert-proxy/clients
```


### Service startup (Systemd)

* You may want to use the supplied Systemd service files from `systemd/`.

  ```
  # chdir to the cloned cert-proxy repo
  systemctl enable $PWD/systemd/cert-proxy-server.service
  ```

  Alternatively you may copy this file to `/etc/systemd/system` and then just
  `systemctl enable cert-proxy-server.service`.

* Start the service

  ```
  systemctl start cert-proxy-server
  ```

## Setup the Client

If you want to use a new Cert-proxy client, you need configure the
server side, as the Cert-proxy server needs to know which client is
allowed to access which private data. (The public data (certificates,
chains, …) is not restricted in access.


### Server side

* Create a client authentication file

  ```shell
  cd /etc/cert-proxy/ca
  bin/mkssl-pem <client-name>
  ```

* Copy now the created `client-name-ssl.pem` and the binary for the
  client to the client system.

  Take the binaries from one of the following places:

  - for linux: `/usr/local/bin/cert-proxy-client`
  - for windows: `/usr/local/bin/windows_amd64/cert-proxy-client.exe`


### Client side

On the client you need to install the above mentioned files
(`<client-name>-ssl.pem`, `cert-proxy-client.exe`) and arrange their
regular startup.  An example startup command line:

```shell
cert-proxy-client -sslfile <client-name>-ssl.pem -certbase <certbase-dir> -verbose
```

This will start the client and download the certs. For more options and
their respective defaults, see the output of

```shell
cert-proxy-client -help
```

To support you, we provide a Systemd timer, and service unit in the
source's `systemd/` directory.


## Daily operation

Repeat starting the Cert-proxy client in sufficient intervals. In
normal operation it downloads (and starts the hooks) only if the files
on the server are fresher than the client's files (based on the file
modification date, *not* on the certificate dates). For Linux systems
you may use the provided Systemd units (`cert-proxy-client.timer` and
`cert-proxy-client.service`).


### New certificates for an existing client

Whenever a new cert should be made available for a client (or a number
of clients), add the new cert domain to the client-config file
`/etc/cert-proxy/clients/<client>`.

Add (line by line) the CNs for certificates the client is allowed to
access. Comments (#) are allowed

No restart required.


## The API

Authentication in both directions is done with X509 certs, issued by a
CA running on the cert-proxy.

Authentication (authn) is required for some operations. Authorization
(authz) for commands exposing private data.

    Requests:
    GET /v1/list                        // authn
    GET /v1/cert/<domain>               // cert, no auth
    GET /v1/chain/<domain>              // chain, no auth
    GET /v1/fullchain/<domain>          // fullchain, no auth
    GET /v1/privkey/<domain>            // privkey, authz required
    GET /v1/bundle/<domain>             // all of the above, authz required¹

Yes, for PEM, multiple requests are used currently.
