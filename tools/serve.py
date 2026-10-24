#!/usr/bin/env python3
"""Serve a directory on localhost with "Cache-Control: no-cache".

Usage: serve.py <directory> <port>

Plain http.server sends no caching headers, so browsers (Safari in
particular) reuse cached app.js and style.css from an earlier build while
loading a new index.html. no-cache makes them revalidate every file, which
costs only a 304 when nothing has changed.

It refuses to start when another server already listens on the port on
127.0.0.1 or ::1. Binding 127.0.0.1 can succeed next to a server on *:port
(IPv6), and browsers then reach that server for localhost, since they try
::1 first.
"""

import functools
import http.server
import socket
import sys


class Handler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header("Cache-Control", "no-cache")
        super().end_headers()


def listening(port):
    """Returns the localhost addresses where a server accepts connections on port."""
    found = []
    for family, host, shown in (
        (socket.AF_INET, "127.0.0.1", "127.0.0.1"),
        (socket.AF_INET6, "::1", "[::1]"),
    ):
        try:
            with socket.socket(family, socket.SOCK_STREAM) as s:
                s.settimeout(0.5)
                s.connect((host, port))
                found.append(shown)
        except OSError:
            pass  # nothing listens there, or no IPv6
    return found


def main():
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    directory, port = sys.argv[1], int(sys.argv[2])
    if busy := listening(port):
        free = next((p for p in range(port + 1, min(port + 100, 65536)) if not listening(p)), None)
        sys.exit(
            f"Port {port} is already in use on {' and '.join(busy)}: http://localhost:{port} "
            f"would reach that server instead.\n"
            f"See which process has it with: lsof -nP -iTCP:{port} -sTCP:LISTEN\n"
            f"Stop it, or use another port" + (f", such as: make serve PORT={free}" if free else ".")
        )
    handler = functools.partial(Handler, directory=directory)
    try:
        server = http.server.ThreadingHTTPServer(("127.0.0.1", port), handler)
    except OSError as e:
        sys.exit(f"Cannot serve on 127.0.0.1:{port}: {e.strerror}")
    with server:
        print(f"Serving {directory} on http://localhost:{port}", flush=True)
        server.serve_forever()


if __name__ == "__main__":
    main()
