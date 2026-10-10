#!/usr/bin/env python3
"""Patch a tester-owned Survival Heroes 3.3.3 APK to use a revival query host.

This tool only changes assets/config.ini. It intentionally does not contain or
redistribute any proprietary game asset. The resulting APK must be zipaligned
and signed before Android will install it.
"""

from __future__ import annotations

import argparse
import configparser
import io
import pathlib
import shutil
import tempfile
import zipfile

CONFIG_PATH = "assets/config.ini"
SIGNATURE_PREFIXES = ("META-INF/",)


def patch_config(raw: bytes, host: str, port: int) -> bytes:
    text = raw.decode("utf-8-sig", errors="strict")
    cfg = configparser.ConfigParser(interpolation=None, strict=False)
    cfg.optionxform = str
    cfg.read_string(text)
    if not cfg.has_section("server_query"):
        raise ValueError("assets/config.ini has no [server_query] section")
    cfg.set("server_query", "Addr", host)
    cfg.set("server_query", "Port", str(port))
    out = io.StringIO()
    cfg.write(out, space_around_delimiters=False)
    return out.getvalue().encode("utf-8")


def patch_apk(source: pathlib.Path, output: pathlib.Path, host: str, port: int) -> None:
    if not host.strip():
        raise ValueError("query host cannot be empty")
    if not 1 <= port <= 65535:
        raise ValueError("port must be between 1 and 65535")

    with zipfile.ZipFile(source, "r") as zin:
        try:
            config = zin.read(CONFIG_PATH)
        except KeyError as exc:
            raise ValueError(f"{CONFIG_PATH} not found in APK") from exc
        patched = patch_config(config, host.strip(), port)

        output.parent.mkdir(parents=True, exist_ok=True)
        with zipfile.ZipFile(output, "w", allowZip64=True) as zout:
            for info in zin.infolist():
                # Old JAR/v1 signature material is invalid after modification.
                # The APK is resigned as a separate build after zipalign.
                upper = info.filename.upper()
                if upper.startswith(SIGNATURE_PREFIXES):
                    continue
                data = patched if info.filename == CONFIG_PATH else zin.read(info.filename)
                clone = zipfile.ZipInfo(info.filename, date_time=info.date_time)
                clone.compress_type = info.compress_type
                clone.comment = info.comment
                clone.extra = info.extra
                clone.internal_attr = info.internal_attr
                clone.external_attr = info.external_attr
                clone.create_system = info.create_system
                clone.flag_bits = info.flag_bits & ~0x08
                zout.writestr(clone, data)


def main() -> int:
    ap = argparse.ArgumentParser(description="Patch Survival Heroes 3.3.3 query host")
    ap.add_argument("apk", type=pathlib.Path)
    ap.add_argument("output", type=pathlib.Path)
    ap.add_argument("--host", required=True, help="revival query-server IP or hostname")
    ap.add_argument("--port", type=int, default=88)
    args = ap.parse_args()
    patch_apk(args.apk, args.output, args.host, args.port)
    print(f"patched {CONFIG_PATH}: Addr={args.host} Port={args.port}")
    print(f"unsigned APK: {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
