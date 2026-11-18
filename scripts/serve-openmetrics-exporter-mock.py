#!/usr/bin/env python3
"""Read-only OpenMetrics mock on loopback (v1.18); not for production scrape."""
import os
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT = ROOT / "docs/development/fixtures/openmetrics-exporter-draft.txt"
PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 19090
BODY_PATH = Path(os.environ.get("COGNARA_OPENMETRICS_PATH", DEFAULT))
BODY = BODY_PATH.read_text()


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path not in ("/metrics", "/metrics/"):
            self.send_response(404)
            self.end_headers()
            return
        data = BODY.encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/plain; version=0.0.4")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *_args):
        return


def main() -> None:
    print(f"OpenMetrics mock on 127.0.0.1:{PORT}/metrics", flush=True)
    HTTPServer(("127.0.0.1", PORT), Handler).serve_forever()


if __name__ == "__main__":
    main()