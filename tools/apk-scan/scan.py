#!/usr/bin/env python3
"""Fast, non-destructive string scanner for APK/XAPK/OBB archives.
It does NOT bypass encryption or protections. It extracts URLs, domains, IPv4 addresses,
and likely networking/configuration keywords from files supplied by the operator.
"""
from __future__ import annotations
import argparse, pathlib, re, zipfile

URL = re.compile(rb'https?://[A-Za-z0-9._~:/?#\[\]@!$&\'()*+,;=%-]{5,}')
DOMAIN = re.compile(rb'(?<![A-Za-z0-9.-])(?:[A-Za-z0-9-]{1,63}\.)+(?:com|net|org|io|cn|game|games|cloud|app)(?![A-Za-z0-9.-])', re.I)
IP = re.compile(rb'(?<!\d)(?:\d{1,3}\.){3}\d{1,3}(?!\d)')
KEYWORDS = (b'login', b'gateway', b'match', b'server', b'protobuf', b'websocket', b'socket', b'udp', b'tcp', b'http')

def blobs(path: pathlib.Path):
    if zipfile.is_zipfile(path):
        with zipfile.ZipFile(path) as z:
            for info in z.infolist():
                if info.is_dir() or info.file_size > 128*1024*1024: continue
                try: yield info.filename, z.read(info)
                except Exception: pass
    else:
        yield path.name, path.read_bytes()

def main():
    ap=argparse.ArgumentParser(); ap.add_argument('file'); ap.add_argument('-o','--output', default='scan-results.txt'); a=ap.parse_args()
    p=pathlib.Path(a.file); found=set(); interesting=[]
    for name,data in blobs(p):
        low=data.lower()
        if any(k in low for k in KEYWORDS): interesting.append(name)
        for rx,label in ((URL,'URL'),(DOMAIN,'DOMAIN'),(IP,'IP')):
            for m in rx.findall(data):
                try: s=m.decode('utf-8','ignore')
                except Exception: continue
                if label=='IP' and any(int(x)>255 for x in s.split('.')): continue
                found.add((label,s))
    out=pathlib.Path(a.output)
    with out.open('w',encoding='utf-8') as f:
        f.write(f'# Scan of {p.name}\n\n')
        for label,s in sorted(found): f.write(f'{label}\t{s}\n')
        f.write('\n# Files containing networking-ish keywords\n')
        for n in sorted(set(interesting)): f.write(n+'\n')
    print(f'wrote {out} ({len(found)} unique network-like strings)')
if __name__=='__main__': main()
