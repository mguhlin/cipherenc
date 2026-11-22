#!/usr/bin/env python3
"""Refresh/check browser integrity and content-addressed asset URLs, without dates."""
from pathlib import Path
import base64, hashlib, json, re, sys
from openpgp_page import build as build_openpgp
root=Path(__file__).resolve().parents[1]/'docs/encrypt'
page=root/'index.html';before=page.read_text()
def local_asset(url):
 name=url.split('?',1)[0]
 file=root/name
 if '://' in name or name.startswith('/') or not file.is_file():return None
 data=file.read_bytes()
 return name,data,f'{name}?v={hashlib.sha256(data).hexdigest()[:12]}'
def script(match):
 tag=match.group(0);src=re.search(r'\bsrc="([^"]+)"',tag)
 if not src:return tag
 asset=local_asset(src[1])
 if not asset:return tag
 name,data,url=asset;tag=tag.replace(src[0],f'src="{url}"')
 tag=re.sub(r'\s+integrity="[^"]+"','',tag)
 digest=base64.b64encode(hashlib.sha384(data).digest()).decode()
 return tag[:-1]+f' integrity="sha384-{digest}">'
def style(match):
 tag=match.group(0);href=re.search(r'\bhref="([^"]+)"',tag)
 if not href:return tag
 asset=local_asset(href[1])
 return tag.replace(href[0],f'href="{asset[2]}"') if asset else tag
after=re.sub(r'<script\b[^>]*>',script,before)
after=re.sub(r'<link\b(?=[^>]*\brel="stylesheet")[^>]*>',style,after)
pgp_page=root/'openpgp.html';pgp_before=pgp_page.read_text() if pgp_page.exists() else '';pgp_after=build_openpgp(after)
manifest=root/'integrity.json';data=json.loads(manifest.read_text())
hashes={name:hashlib.sha256(after.encode() if name=='index.html' else pgp_after.encode() if name=='openpgp.html' else (root/name).read_bytes()).hexdigest() for name in data['sha256']}
output=json.dumps({'sha256':hashes},indent=2)+'\n'
landing=root.parent/'index.html';landing_before=landing.read_text()
landing_after=re.sub(r'href="encrypt/(?:\?v=[a-f0-9]{12})?(#[^" ]*)?"',lambda match:'href="encrypt/?v='+hashes['index.html'][:12]+(match[1] or '')+'"',landing_before)
landing_after=re.sub(r'href="site\.css(?:\?v=[a-f0-9]{12})?"', 'href="site.css?v='+hashlib.sha256((root.parent/'site.css').read_bytes()).hexdigest()[:12]+'"',landing_after)
if '--check' in sys.argv:
 if before!=after or manifest.read_text()!=output or landing_before!=landing_after or pgp_before!=pgp_after:raise SystemExit('Browser integrity records need regeneration: python3 scripts/update-integrity.py')
 print('Browser integrity records and content-addressed asset URLs match')
else:page.write_text(after);manifest.write_text(output);landing.write_text(landing_after);pgp_page.write_text(pgp_after)
