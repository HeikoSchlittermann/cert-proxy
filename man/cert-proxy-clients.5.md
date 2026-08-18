% cert-proxy-clients 5 "August 2026" "cert-proxy" "File Formats"

# NAME

cert-proxy-clients - per-client authorization files for cert-proxy-server

# SYNOPSIS

*/etc/cert-proxy/clients/CN*

# DESCRIPTION

**cert-proxy-server**(8) authorizes access to private keys and PKCS12 bundles per
client. For a request that needs authorization the server takes the common name
of the presented client certificate, opens the file of that exact name in the
directory given by its **-ccd** option, and grants the request only if the
requested domain is listed in that file.

There is one file per client. A missing file, or a common name that is not a
valid file name, denies every authorized request from that client; the server
answers **401 Unauthorized** and does not disclose which of the two applied. The
files are read on each request, so edits take effect immediately and no reload is
needed.

The public endpoints of **cert-proxy**(7) are not covered by these files. They
require no client certificate at all, and listing a domain here does not make it
any more or less public.

# FILE NAME

The file name is the client certificate's common name, used verbatim. To keep the
lookup safe on every supported platform, a common name is accepted only if it
matches the regular expression

```
^[a-zA-Z0-9._-]+$
```

contains no empty label - that is, no leading dot, trailing dot or double dot -
and its first label is not one of the DOS device names such as *CON*, *PRN*,
*AUX*, *NUL*, *COM1* or *LPT1*. The wildcard character is rejected as well.
Because the path separator is not in the accepted set, a common name can never
escape the directory.

# FORMAT

The file is read line by line and holds one domain per line.

On each line, everything from the first **#** to the end of the line is a comment
and is discarded. Spaces, tabs and carriage returns are then removed from both
ends of what remains. If nothing remains, the line is skipped. This means blank
lines and whole-line comments are permitted, a comment may follow a domain on the
same line, and CRLF files are read correctly.

Entries are compared literally against the requested domain. There is no
wildcard, prefix or suffix matching: a client authorized for *example.com* is not
thereby authorized for *www.example.com*.

The same format is accepted by the **-cnfile** option of
**cert-proxy-client**(8), where the lines name the domains to fetch rather than
the domains to allow.

# EXAMPLES

Authorize one host for its own name and the bare domain:

```
# /etc/cert-proxy/clients/www.example.com
example.com
www.example.com        # the vhost itself
```

Authorize a mail host, keeping a retired name on record:

```
# /etc/cert-proxy/clients/mail.example.com
mail.example.com
imap.example.com
# smtp.example.com     # decommissioned 2026-03
```

# SEE ALSO

**cert-proxy**(7), **cert-proxy-server**(8), **cert-proxy-client**(8)

# AUTHORS

Heiko Schlittermann <hs@schlittermann.de>
