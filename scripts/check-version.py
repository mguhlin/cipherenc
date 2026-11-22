#!/usr/bin/env python3
"""Check that app, website and Android identify the same release."""
from pathlib import Path
import re,xml.etree.ElementTree as ET
root=Path(__file__).resolve().parents[1]
version=re.search(r'const version = "([^"]+)"',(root/'main.go').read_text())[1]
assert re.fullmatch(r'\d+\.\d+\.\d+',version)
app=(root/'docs/encrypt/app.js').read_text()
assert re.search(r"APP_VERSION='([^']+)'",app)[1]==version,'Browser version differs'
for name in ['index.html','openpgp.html']:
 assert f'CipherEnc {version} · Browser version' in (root/'docs/encrypt'/name).read_text(),name
landing=(root/'docs/index.html').read_text()
assert f'Latest version: {version}' in landing,'Website release label differs'
assert all(re.search(r'/v([^/]+)(?:/|$)',url)[1]==version for url in re.findall(r'https://github.com/mguhlin/cipherenc/releases/(?:download|tag)/[^" ]+',landing)),'Website release links differ'
manifest=ET.parse(root/'android/AndroidManifest.xml').getroot();ns='{http://schemas.android.com/apk/res/android}'
assert manifest.attrib[ns+'versionName']==version+'-evaluation','Android version differs'
assert manifest.find('application').attrib[ns+'label']=='CipherEnc '+version,'Android launcher version differs'
assert f'**Current version: {version}.**' in (root/'README.md').read_text(),'README version differs'
print('App, website, downloads and Android consistently identify CipherEnc '+version)
