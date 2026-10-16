#!/usr/bin/env python3
"""Remote Write endpoint that rejects every upload with HTTP 500.

Usage (on the VM, with the proxy environment scrubbed):

    nohup env -u HTTP_PROXY -u HTTPS_PROXY -u http_proxy -u https_proxy \
        python3 http500.py > http500.log 2>&1 &

Listens on 127.0.0.1:7001. Point api.heartbeat.uri at
http://127.0.0.1:7001/api/v1/write to exercise the rejection scenario:
the heartbeat must report the 500 and keep the pending heartbeat.
"""

import sys

from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class RejectingHandler(BaseHTTPRequestHandler):
    """Answers every POST with 500 after draining the request body."""

    def do_POST(self):
        """Consume the request body and respond with 500."""
        length = int(self.headers.get("Content-Length", 0))
        self.rfile.read(length)
        self.send_response(500)
        self.send_header("Content-Length", "0")
        self.end_headers()

    def log_message(self, fmt, *args):
        """Log to stderr so nohup captures the request trail."""
        sys.stderr.write("HTTP500: " + (fmt % args) + "\n")


if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", 7001), RejectingHandler).serve_forever()
