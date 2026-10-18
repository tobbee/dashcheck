#!/usr/bin/env python3
"""Build the inventory of DASH-IF Conformance tool checks.

Usage: dashif-inventory.py <DASH-IF-Conformance clone> <docs/dashif-checks.json>

Extracts the Schematron asserts (schematron/*.sch) and the module checks
(->add(section: ..., test: ...) in app/Modules) and writes them to the JSON
file. Status, rules and notes of checks already in the file are kept, matched
on source and text, so the script can be rerun when the upstream tool changes.
"""

import html
import json
import os
import re
import subprocess
import sys

SCH_FILES = [("schematron.sch", "SCH"), ("dvb-dash.sch", "SCH-DVB"), ("ac4-generic.sch", "SCH-AC4")]
MODULE_PREFIX = {
    "Common": "COMMON", "IOP": "IOP", "CMAF": "CMAF", "LowLatency": "LL", "DVB": "DVB",
    "HbbTV": "HBBTV", "CTAWave": "WAVE", "WaveHLSInterop": "WAVEHLS", "Dolby": "DOLBY",
}
STR = r'''"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*\''''
ADD_RE = re.compile(
    r"section:\s*(" + STR + r"|\$[\w>\-]+)(?:\s*\.\s*(?:" + STR + r"))*\s*,\s*test:\s*(" + STR + r")"
    r"((?:\s*\.\s*(?:" + STR + r"|\$[A-Za-z_>\-\[\]'\"0-9()]+))*)", re.S)


def unquote(s):
    return s[1:-1].replace('\\"', '"').replace("\\'", "'")


def clean(s):
    return re.sub(r"\s+", " ", s).strip()


def schematron(root):
    for fname, prefix in SCH_FILES:
        text = open(os.path.join(root, "schematron", fname), encoding="utf-8").read()
        text = re.sub(r"<!--.*?-->", "", text, flags=re.S)
        n = 0
        rules = re.finditer(r'<(?:sch:)?rule\s+context="([^"]*)"[^>]*>(.*?)</(?:sch:)?rule>', text, re.S)
        for rule in rules:
            context = html.unescape(rule.group(1))
            asserts = re.finditer(r'<(?:sch:)?assert\s+test="([^"]*)"[^>]*>(.*?)</(?:sch:)?assert>', rule.group(2), re.S)
            for a in asserts:
                n += 1
                msg = clean(html.unescape(re.sub(r"<[^>]+>", "", a.group(2))))
                yield {"id": f"{prefix}-{n}", "source": f"schematron/{fname}", "context": context, "text": msg}


def modules(root):
    base = os.path.join(root, "app", "Modules")
    counts = {}
    for module in sorted(os.listdir(base)):
        mdir = os.path.join(base, module)
        if not os.path.isdir(mdir):
            continue
        prefix = MODULE_PREFIX.get(module, module.upper())
        for dirpath, _, files in sorted(os.walk(mdir)):
            for fname in sorted(files):
                if not fname.endswith(".php"):
                    continue
                path = os.path.join(dirpath, fname)
                text = open(path, encoding="utf-8").read()
                for m in ADD_RE.finditer(text):
                    section = unquote(m.group(1)) if m.group(1)[0] in "\"'" else ""
                    test = clean(unquote(m.group(2)))
                    if m.group(3).strip():
                        test += " …"
                    counts[prefix] = counts.get(prefix, 0) + 1
                    yield {
                        "id": f"{prefix}-{counts[prefix]}",
                        "source": os.path.relpath(path, base),
                        "context": section,
                        "text": test,
                    }


def main():
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    root, out = sys.argv[1], sys.argv[2]
    old = {}
    if os.path.exists(out):
        with open(out, encoding="utf-8") as f:
            for c in json.load(f)["checks"]:
                old[(c["source"], c["text"])] = c
    commit = subprocess.run(["git", "-C", root, "log", "-1", "--format=%h %cs"],
                            capture_output=True, text=True, check=True).stdout.split()
    checks = []
    for c in list(schematron(root)) + list(modules(root)):
        prev = old.pop((c["source"], c["text"]), None)
        c["status"] = prev["status"] if prev else "todo"
        c["rules"] = prev.get("rules", []) if prev else []
        c["note"] = prev.get("note", "") if prev else ""
        checks.append(c)
    for (source, text) in old:
        print(f"removed upstream: {source}: {text}", file=sys.stderr)
    inv = {
        "upstream": {"repo": "https://github.com/Dash-Industry-Forum/DASH-IF-Conformance",
                     "commit": commit[0], "date": commit[1]},
        "checks": checks,
    }
    with open(out, "w", encoding="utf-8") as f:
        json.dump(inv, f, indent=1, ensure_ascii=False)
        f.write("\n")
    print(f"{len(checks)} checks written to {out}", file=sys.stderr)


if __name__ == "__main__":
    main()
