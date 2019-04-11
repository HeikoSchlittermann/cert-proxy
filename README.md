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
       cert-proxy
         | |
	 ^ v
	 | |
       cert-proxy-client

As the cert-proxy may send sensitive information to the client, the
client has to authenticate itself. As the client has to trust the
information from the proxy, the server needs to authenticate with the
client.


## Operation

### Setup the server

First install the binaries.  For authentication betwenn client and proxy
X509 is used (in both directions). Client and Proxy use a single file
containing the information they need (default name: ssl.pem). You're
encouraged to use the provided minimalistic CA.

First create the CA:

    cd CA
    cp lib/vars.sh.example
    <edit> lib/.vars.sh
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

    Requests:
    GET /cert/<cn>	    // PEM
    GET /chain/<cn>	    // PEM
    GET /privkey/<cn>	    // PEM
    GET /fullchain/<cn>	    // PEM
    GET /pkcs12/<cn>

Yes, for PEM, multiple requests are used.
