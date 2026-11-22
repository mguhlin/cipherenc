#!/usr/bin/env python3
"""Refresh/check shipped browser script integrity and source hashes, without timestamps."""
from pathlib import Path
import base64, hashlib, json, re, sys
root=Path(__file__).resolve().parents[1]/'docs/encrypt'
page=root/'index.html';before=page.read_text()
def script(match):
 name=match.group(1)
 digest=base64.b64encode(hashlib.sha384((root/name).read_bytes()).digest()).decode()
 return f'<script src="{name}" integrity="sha384-{digest}"></script>'
after=re.sub(r'<script src="([^"]+)"(?: integrity="[^"]+")?></script>',script,before)
manifest=root/'integrity.json';data=json.loads(manifest.read_text());hashes={}
for name in data['sha256']:
 hashes[name]=hashlib.sha256(after.encode() if name=='index.html' else (root/name).read_bytes()).hexdigest()
output=json.dumps({'sha256':hashes},indent=2)+'\n'
if '--check' in sys.argv:
 if before!=after or manifest.read_text()!=output:raise SystemExit('Browser integrity records need regeneration: python3 scripts/update-integrity.py')
 print('Browser integrity records match shipped files')
else:page.write_text(after);manifest.write_text(output)
