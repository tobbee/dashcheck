#!/usr/bin/env python3
"""Serve a directory on localhost with "Cache-Control: no-cache".

Usage: serve.py <directory> <port>

Plain http.server sends no caching headers, so browsers (Safari in
particular) reuse cached app.js and style.css from an earlier build while
loading a new index.html. no-cache makes them revalidate every file, which
costs only a 304 when nothing has changed.
"""

import functools
import http.server
import sys


class Handler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header("Cache-Control", "no-cache")
        super().end_headers()


def main():
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    directory, port = sys.argv[1], int(sys.argv[2])
    handler = functools.partial(Handler, directory=directory)
    with http.server.ThreadingHTTPServer(("127.0.0.1", port), handler) as server:
        print(f"Serving {directory} on http://localhost:{port}", flush=True)
        server.serve_forever()


if __name__ == "__main__":
    main()
