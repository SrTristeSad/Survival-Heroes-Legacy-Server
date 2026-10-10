#!/usr/bin/env python3
"""Search decompressed PCK0 resources without writing proprietary files to disk."""

from __future__ import annotations

import argparse
import pathlib
import re
import sys

# Keep this standalone-friendly when invoked from repository root.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from pck0_extract import load_index, read_entry  # noqa: E402


def main() -> int:
    ap = argparse.ArgumentParser(description="grep decoded resources inside a PCK0 v15 archive")
    ap.add_argument("archive", type=pathlib.Path)
    ap.add_argument("pattern", help="case-insensitive regex searched in decoded text")
    ap.add_argument("--path", default=".*", help="case-insensitive regex filtering archive paths")
    ap.add_argument("--context", type=int, default=2, help="lines of context around each match")
    ap.add_argument("--max-file-size", type=int, default=8 * 1024 * 1024)
    ap.add_argument("--limit", type=int, default=500, help="maximum match groups emitted")
    args = ap.parse_args()

    content_re = re.compile(args.pattern, re.IGNORECASE)
    path_re = re.compile(args.path, re.IGNORECASE)
    _, entries = load_index(args.archive)

    emitted = 0
    scanned = 0
    failures = 0
    with args.archive.open("rb") as fp:
        for entry in entries:
            if entry.size > args.max_file_size or not path_re.search(entry.path):
                continue
            scanned += 1
            try:
                raw = read_entry(fp, entry)
            except Exception as exc:
                failures += 1
                print(f"ERROR\t{entry.path}\t{exc}", file=sys.stderr)
                continue

            # Configuration resources are mostly UTF-8/ASCII/GB-family text.
            # latin-1 is a final byte-preserving fallback useful for key search.
            text = None
            for enc in ("utf-8-sig", "gb18030", "utf-16", "latin-1"):
                try:
                    candidate = raw.decode(enc)
                except UnicodeDecodeError:
                    continue
                if "\x00" not in candidate[:512] or enc == "utf-16":
                    text = candidate
                    break
            if text is None:
                text = raw.decode("latin-1", errors="replace")

            lines = text.splitlines()
            hit_lines = [i for i, line in enumerate(lines) if content_re.search(line)]
            for i in hit_lines:
                lo = max(0, i - args.context)
                hi = min(len(lines), i + args.context + 1)
                print(f"===== {entry.path} : line {i + 1} =====")
                for n in range(lo, hi):
                    mark = ">" if n == i else " "
                    # Bound pathological/binary-looking lines.
                    line = lines[n].replace("\x00", "\\0")
                    if len(line) > 1000:
                        line = line[:1000] + "…"
                    print(f"{mark}{n + 1}: {line}")
                emitted += 1
                if emitted >= args.limit:
                    print(f"scanned={scanned} failures={failures} emitted={emitted}", file=sys.stderr)
                    return 0

    print(f"scanned={scanned} failures={failures} emitted={emitted}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
