#!/usr/bin/env python3
"""Read Snail Games PCK0 archives used by Survival Heroes 3.3.3.

This tool intentionally does not ship any game data. It lets an owner of a
compatible client inspect or extract files from their own OBB/PCK archive.

Observed Survival Heroes v15 layout (little endian):

    0x00  char[4]  magic = PCK0
    0x04  u32      version (15)
    0x08  u16      reserved/flags
    0x0a  u32      file_count
    0x0e  u32      data_offset
    0x12  u8       reserved

Each v15 index entry starts with a u16 record length, followed by:

    u64 data_offset
    u32 unpacked_size
    u32 packed_size (0 means stored/raw)
    u32 hash/check value
    u8[5] metadata (observed in v15)
    char[] zero-terminated path

The record-length field is used as the authority so malformed entries fail
closed instead of letting a path walk into the next record.
"""

from __future__ import annotations

import argparse
import csv
import dataclasses
import os
import pathlib
import re
import struct
import sys
import zlib
from typing import BinaryIO, Iterable, Iterator, Optional


HEADER = struct.Struct("<4sIHIIB")
# record_len is separate; fields below total 25 bytes for v15 before filename.
ENTRY_FIXED = struct.Struct("<QIII5s")


@dataclasses.dataclass(frozen=True)
class PckHeader:
    version: int
    flags: int
    file_count: int
    data_offset: int
    reserved: int


@dataclasses.dataclass(frozen=True)
class PckEntry:
    index: int
    record_offset: int
    record_length: int
    offset: int
    size: int
    packed_size: int
    hash_value: int
    metadata: bytes
    path: str

    @property
    def stored_size(self) -> int:
        return self.packed_size if self.packed_size else self.size

    @property
    def compressed(self) -> bool:
        return bool(self.packed_size and self.packed_size != self.size)


def read_exact(fp: BinaryIO, n: int) -> bytes:
    data = fp.read(n)
    if len(data) != n:
        raise EOFError(f"expected {n} bytes, got {len(data)}")
    return data


def read_header(fp: BinaryIO) -> PckHeader:
    raw = read_exact(fp, HEADER.size)
    magic, version, flags, file_count, data_offset, reserved = HEADER.unpack(raw)
    if magic != b"PCK0":
        raise ValueError(f"not a PCK0 archive (magic={magic!r})")
    if file_count > 5_000_000:
        raise ValueError(f"implausible file count: {file_count}")
    return PckHeader(version, flags, file_count, data_offset, reserved)


def decode_path(raw: bytes) -> str:
    raw = raw.split(b"\x00", 1)[0]
    for encoding in ("utf-8", "gb18030", "latin-1"):
        try:
            return raw.decode(encoding)
        except UnicodeDecodeError:
            pass
    return raw.decode("latin-1", errors="replace")


def iter_entries(fp: BinaryIO, header: PckHeader) -> Iterator[PckEntry]:
    if header.version != 15:
        raise NotImplementedError(
            f"this parser currently targets PCK0 v15; archive says v{header.version}"
        )

    for index in range(header.file_count):
        record_offset = fp.tell()
        raw_len = fp.read(2)
        if len(raw_len) != 2:
            raise EOFError(f"entry {index}: truncated record length")
        (record_len,) = struct.unpack("<H", raw_len)
        if record_len < 2 + ENTRY_FIXED.size + 2:
            raise ValueError(f"entry {index}: implausible record length {record_len}")

        body_len = record_len - 2
        body = read_exact(fp, body_len)
        if len(body) < ENTRY_FIXED.size + 1:
            raise ValueError(f"entry {index}: record too short")

        offset, size, packed_size, hash_value, metadata = ENTRY_FIXED.unpack_from(body, 0)
        path_blob = body[ENTRY_FIXED.size:]
        nul = path_blob.find(b"\x00")
        if nul < 0:
            raise ValueError(f"entry {index}: path is not NUL-terminated")
        path = decode_path(path_blob[: nul + 1])
        if not path:
            raise ValueError(f"entry {index}: empty path")

        yield PckEntry(
            index=index,
            record_offset=record_offset,
            record_length=record_len,
            offset=offset,
            size=size,
            packed_size=packed_size,
            hash_value=hash_value,
            metadata=metadata,
            path=path,
        )


def load_index(archive: pathlib.Path) -> tuple[PckHeader, list[PckEntry]]:
    with archive.open("rb") as fp:
        header = read_header(fp)
        entries = list(iter_entries(fp, header))
    return header, entries


def safe_relative_path(name: str) -> pathlib.Path:
    name = name.replace("\\", "/")
    # Strip Windows drive/UNC-ish prefixes and traversal.
    name = re.sub(r"^[A-Za-z]:", "", name).lstrip("/")
    parts = [p for p in name.split("/") if p not in ("", ".", "..")]
    if not parts:
        raise ValueError(f"unsafe/empty archive path: {name!r}")
    return pathlib.Path(*parts)


def decompress_payload(data: bytes, expected_size: int) -> bytes:
    errors: list[str] = []
    for wbits, label in ((zlib.MAX_WBITS, "zlib"), (-zlib.MAX_WBITS, "raw-deflate")):
        try:
            out = zlib.decompress(data, wbits)
            if expected_size and len(out) != expected_size:
                errors.append(f"{label}: size {len(out)} != {expected_size}")
                continue
            return out
        except zlib.error as exc:
            errors.append(f"{label}: {exc}")
    raise ValueError("unable to decompress payload; " + "; ".join(errors))


def read_entry(fp: BinaryIO, entry: PckEntry) -> bytes:
    fp.seek(entry.offset)
    payload = read_exact(fp, entry.stored_size)
    if entry.packed_size == 0 or entry.packed_size == entry.size:
        return payload
    return decompress_payload(payload, entry.size)


def write_index(entries: Iterable[PckEntry], out: BinaryIO | object) -> None:
    writer = csv.writer(out, delimiter="\t", lineterminator="\n")
    writer.writerow(
        [
            "index",
            "path",
            "offset",
            "size",
            "packed_size",
            "compressed",
            "hash_hex",
            "metadata_hex",
            "record_offset",
            "record_length",
        ]
    )
    for e in entries:
        writer.writerow(
            [
                e.index,
                e.path,
                e.offset,
                e.size,
                e.packed_size,
                int(e.compressed),
                f"{e.hash_value:08x}",
                e.metadata.hex(),
                e.record_offset,
                e.record_length,
            ]
        )


def compile_match(pattern: Optional[str]) -> Optional[re.Pattern[str]]:
    if not pattern:
        return None
    return re.compile(pattern, re.IGNORECASE)


def selected(entries: Iterable[PckEntry], pattern: Optional[re.Pattern[str]]) -> Iterator[PckEntry]:
    for e in entries:
        if pattern is None or pattern.search(e.path):
            yield e


def command_info(args: argparse.Namespace) -> int:
    header, entries = load_index(args.archive)
    total_stored = sum(e.stored_size for e in entries)
    compressed = sum(1 for e in entries if e.compressed)
    print(f"version={header.version}")
    print(f"flags={header.flags}")
    print(f"file_count={header.file_count}")
    print(f"data_offset={header.data_offset}")
    print(f"reserved={header.reserved}")
    print(f"indexed_entries={len(entries)}")
    print(f"compressed_entries={compressed}")
    print(f"stored_bytes={total_stored}")
    return 0


def command_list(args: argparse.Namespace) -> int:
    _, entries = load_index(args.archive)
    pattern = compile_match(args.match)
    rows = list(selected(entries, pattern))
    if args.limit is not None:
        rows = rows[: args.limit]
    if args.tsv:
        write_index(rows, sys.stdout)
    else:
        for e in rows:
            print(e.path)
    return 0


def command_extract(args: argparse.Namespace) -> int:
    _, entries = load_index(args.archive)
    pattern = compile_match(args.match)
    rows = list(selected(entries, pattern))
    if args.limit is not None:
        rows = rows[: args.limit]

    out_root = args.output.resolve()
    out_root.mkdir(parents=True, exist_ok=True)
    failures = 0
    extracted = 0

    with args.archive.open("rb") as fp:
        for entry in rows:
            try:
                rel = safe_relative_path(entry.path)
                target = (out_root / rel).resolve()
                if out_root != target and out_root not in target.parents:
                    raise ValueError("target escaped output directory")
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(read_entry(fp, entry))
                extracted += 1
                if args.verbose:
                    print(f"extracted\t{entry.path}\t{entry.size}")
            except Exception as exc:  # keep batch extraction useful
                failures += 1
                print(f"ERROR\t{entry.path}\t{exc}", file=sys.stderr)
                if args.fail_fast:
                    return 2

    print(f"extracted={extracted} failures={failures}", file=sys.stderr)
    return 1 if failures else 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Inspect/extract Survival Heroes PCK0 v15 archives")
    sub = parser.add_subparsers(dest="command", required=True)

    p_info = sub.add_parser("info", help="print archive metadata")
    p_info.add_argument("archive", type=pathlib.Path)
    p_info.set_defaults(func=command_info)

    p_list = sub.add_parser("list", help="list archive entries")
    p_list.add_argument("archive", type=pathlib.Path)
    p_list.add_argument("--match", help="case-insensitive regex applied to archive path")
    p_list.add_argument("--limit", type=int)
    p_list.add_argument("--tsv", action="store_true", help="emit metadata TSV instead of only paths")
    p_list.set_defaults(func=command_list)

    p_extract = sub.add_parser("extract", help="extract selected files")
    p_extract.add_argument("archive", type=pathlib.Path)
    p_extract.add_argument("output", type=pathlib.Path)
    p_extract.add_argument("--match", help="case-insensitive regex applied to archive path")
    p_extract.add_argument("--limit", type=int)
    p_extract.add_argument("--verbose", action="store_true")
    p_extract.add_argument("--fail-fast", action="store_true")
    p_extract.set_defaults(func=command_extract)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    try:
        return int(args.func(args))
    except (OSError, EOFError, ValueError, NotImplementedError) as exc:
        parser.error(str(exc))
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
