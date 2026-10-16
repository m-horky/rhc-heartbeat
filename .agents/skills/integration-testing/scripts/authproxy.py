#!/usr/bin/env python3
"""HTTP proxy enforcing Basic proxy authentication.

Valid credentials are "good:pass". Authenticated requests are forwarded to
their absolute-URI target; everything else gets a 407 with a
Proxy-Authenticate challenge.

Usage (on the VM, with the proxy environment scrubbed):

    nohup env -u HTTP_PROXY -u HTTPS_PROXY -u http_proxy -u https_proxy \
        python3 authproxy.py > authproxy.log 2>&1 &

Listens on 127.0.0.1:7000. Configure the heartbeat with:

    [http.proxy]
    uri = "http://good:pass@127.0.0.1:7000"   # or bad:pass for the 407 case

    [api.heartbeat]
    uri = "http://<non-loopback-host>:<port>/api/v1/write"

The endpoint MUST NOT be a loopback address: Go's httpproxy bypasses
localhost/127.0.0.0-8 regardless of the configured proxy, which silently
invalidates the scenario. Use the VM's external IP or the Prometheus
container IP (podman inspect prometheus --format
'{{.NetworkSettings.IPAddress}}', port 9090 inside the container).
"""

import base64
import sys
import urllib.error
import urllib.request

from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

CREDENTIALS = "good:pass"


class AuthenticatingProxy(BaseHTTPRequestHandler):
    """Forwards authorized absolute-URI POSTs, rejects the rest with 407."""

    protocol_version = "HTTP/1.1"

    def do_POST(self):
        """Check Proxy-Authorization, then forward or challenge."""
        # Drain the request body before replying, even on 407. If the body
        # is left unread, the keep-alive connection misinterprets it as the
        # next request line and the Go client logs an "unsolicited
        # response" warning that pollutes the scenario output.
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length)

        expected = "Basic " + base64.b64encode(CREDENTIALS.encode()).decode()
        if self.headers.get("Proxy-Authorization", "") != expected:
            self.send_response(407)
            self.send_header("Proxy-Authenticate", 'Basic realm="test"')
            self.send_header("Content-Length", "0")
            self.end_headers()
            return

        request = urllib.request.Request(self.path, data=body, method="POST")
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
        sys.stderr.write("PROXY: " + (fmt % args) + "\n")


if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", 7000), AuthenticatingProxy).serve_forever()
