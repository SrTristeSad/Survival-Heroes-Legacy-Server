#!/usr/bin/env python3
from pathlib import Path
import re

sections=[]
for line in Path('reports/sections.txt').read_text(errors='replace').splitlines():
    m=re.match(r'\s*\[\s*\d+\]\s+(\S+)\s+\S+\s+([0-9a-fA-F]+)\s+([0-9a-fA-F]+)\s+([0-9a-fA-F]+)', line)
    if m:
        name,addr,off,size=m.groups()
        sections.append((name,int(addr,16),int(off,16),int(size,16)))

targets=[]
for line in Path('reports/login-success-strings.txt').read_text(errors='replace').splitlines():
    m=re.match(r'\s*([0-9a-fA-F]+)\s+(.*)', line)
    if not m:
        continue
    foff=int(m.group(1),16); text=m.group(2); va=None
    for _,addr,off,size in sections:
        if off <= foff < off+size:
            va=addr+(foff-off); break
    if va is not None:
        targets.append((va,text,foff))

lines=Path('work/disasm.txt').read_text(errors='replace').splitlines()
by_page={}
for i,line in enumerate(lines):
    m=re.search(r'adrp\s+(x\d+),\s*([0-9a-fA-F]+)',line)
    if m:
        by_page.setdefault(int(m.group(2),16),[]).append((i,m.group(1)))

out=[]; xaddrs=[]
for va,text,foff in targets:
    page=va & ~0xfff; loff=va & 0xfff
    for i,reg in by_page.get(page,[]):
        for j in range(i+1,min(i+10,len(lines))):
            mm=re.search(rf'add\s+{reg},\s*{reg},\s*#0x([0-9a-fA-F]+)',lines[j])
            if mm and int(mm.group(1),16)==loff:
                am=re.match(r'\s*([0-9a-fA-F]+):',lines[i])
                if am:
                    a=int(am.group(1),16); xaddrs.append(a)
                    out.append(f'===== XREF 0x{a:x} -> VA 0x{va:x} file 0x{foff:x}: {text} =====')
                    out.extend(lines[max(0,i-90):min(len(lines),j+220)])
                break
Path('reports/login-success-xrefs.txt').write_text('\n'.join(out),encoding='utf-8')
Path('reports/xref-addrs.txt').write_text('\n'.join(hex(x) for x in sorted(set(xaddrs))),encoding='utf-8')
print(f'targets={len(targets)} xrefs={len(set(xaddrs))}')
