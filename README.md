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

Authentication in both directions is done with X509 certs, issued by a
CA running on the cert-proxy.


    Requests:
    GET /crt/<cn>
    GET /privkey/<cn>
    GET /fullchain/<cn>

## operation

Start the proxy on the proxy host:

    cert-proxy

Run the client:

    cert-proxy-client -interval 1h <dn>...
