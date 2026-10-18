#!/usr/bin/env python3
"""Set the status of DASH-IF checks in docs/dashif-checks.json.

Usage: set-status.py <updates.json | ->

The updates map check IDs to [status, [rules], note], for example
  {"DVB-71": ["done", ["SEG-DVB-TRAF"], ""],
   "DVB-32": ["n/a", [], "a permissive (MAY) statement"]}
Read from a file, or from stdin with "-". Statuses: done, partial, todo, n/a.
Afterwards run: go test ./internal/coverage -update
"""

import json
import os
import sys

INVENTORY = os.path.join(os.path.dirname(__file__), "..", "docs", "dashif-checks.json")
STATUSES = {"done", "partial", "todo", "n/a"}


def main():
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    src = sys.stdin if sys.argv[1] == "-" else open(sys.argv[1], encoding="utf-8")
    updates = json.load(src)
    with open(INVENTORY, encoding="utf-8") as f:
        inv = json.load(f)
    checks = {c["id"]: c for c in inv["checks"]}
    for cid, (status, rules, note) in updates.items():
        if cid not in checks:
            sys.exit(f"unknown check ID {cid}")
        if status not in STATUSES:
            sys.exit(f"{cid}: unknown status {status}")
        checks[cid].update(status=status, rules=rules, note=note)
    with open(INVENTORY, "w", encoding="utf-8") as f:
        json.dump(inv, f, indent=1, ensure_ascii=False)
        f.write("\n")
    print(f"updated {len(updates)} checks", file=sys.stderr)


if __name__ == "__main__":
    main()
