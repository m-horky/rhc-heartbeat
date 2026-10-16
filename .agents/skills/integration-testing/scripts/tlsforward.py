#!/usr/bin/env python3
"""TLS Remote Write endpoint forwarding to the local Prometheus.

Serves HTTPS on 127.0.0.1:7443 and forwards every POST body to
http://127.0.0.1:9000/api/v1/write, replaying the response status. Used for
the TLS verification scenarios: run it once with a certificate signed by the
configured authority (upload must succeed) and once with an untrusted
certificate (client must report a TLS error and keep the pending heartbeat).

The certificate and key are selected with the TLS_CERT and TLS_KEY
environment variables. The server certificate MUST carry a subjectAltName
matching the address in api.heartbeat.uri (e.g. subjectAltName=IP:127.0.0.1):
Go rejects CN-only certificates. Generate the material along these lines:

    openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
        -keyout ca.key -out ca.pem -subj "/CN=Test CA"
    openssl req -newkey rsa:2048 -nodes -keyout server.key \
        -out server.csr -subj "/CN=127.0.0.1"
    printf "subjectAltName=IP:127.0.0.1" > san.ext
    openssl x509 -req -in server.csr -CA ca.pem -CAkey ca.key \
        -out server-trusted.pem -days 1 -extfile san.ext
    # untrusted variant: a self-signed certificate with the same SAN
    openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
        -keyout rogue.key -out rogue.pem -subj "/CN=rogue" \
        -addext "subjectAltName=IP:127.0.0.1"

Usage (on the VM, with the proxy environment scrubbed):

    nohup env -u HTTP_PROXY -u HTTPS_PROXY -u http_proxy -u https_proxy \
        TLS_CERT=server-trusted.pem TLS_KEY=server.key \
        python3 tlsforward.py > tlsforward.log 2>&1 &

The CA is configured on the client side through rhsm.conf, section [rhsm],
key repo_ca_cert (NOT through rhc-heartbeat.conf, whose unknown keys are
silently ignored).
"""

import os
import ssl
import sys
import urllib.error
import urllib.request

from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

UPSTREAM = "http://127.0.0.1:9000/api/v1/write"


class ForwardingHandler(BaseHTTPRequestHandler):
    """Replays POST bodies to the upstream Prometheus receiver."""

    protocol_version = "HTTP/1.1"

    def do_POST(self):
        """Forward the request body and mirror the upstream status."""
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length)
        request = urllib.request.Request(UPSTREAM, data=body, method="POST")
        for header in ("Content-Type", "Content-Encoding"):
            if header in self.headers:
                request.add_header(header, self.headers[header])
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                response.read()
                status = response.status
        except urllib.error.HTTPError as error:
            error.read()
            status = error.code
        self.send_response(status)
        self.send_header("Content-Length", "0")
        self.end_headers()

    def log_message(self, fmt, *args):
        """Log to stderr so nohup captures the request trail."""
        sys.stderr.write("TLSFWD: " + (fmt % args) + "\n")


if __name__ == "__main__":
    server = ThreadingHTTPServer(("127.0.0.1", 7443), ForwardingHandler)
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(os.environ["TLS_CERT"], os.environ["TLS_KEY"])
    server.socket = context.wrap_socket(server.socket, server_side=True)
    server.serve_forever()
