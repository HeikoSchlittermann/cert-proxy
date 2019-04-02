# CERT-PROXY

cert-proxy should be the trustworthy system acting on behalf of cert-proxy
clients. It interacts with the Let's Encrypt CA system as a ACME
protocol client. Certproxy is responsible for responding to the
challenges from LE.

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

As the cert-proxy may send sensitive information to the
client, the client has to authenticate itself.
