#!/usr/bin/env python3
"""Build a development-signed, offline Android evaluation APK without Gradle.

The Android SDK platform/build-tools archives are official Google artifacts.
Toolchain and signing material stay outside this repository and its source ZIP.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import secrets
import re
import shutil
import subprocess
import urllib.request
import zipfile

ROOT = Path(__file__).resolve().parent
PROJECT = ROOT.parent
VERSION = re.search(r'const version = "([^"]+)"', (PROJECT/'main.go').read_text())[1]
ARTIFACTS = (
    ('platform-36_r02.zip', '2c1a80dd4d9f7d0e6dd336ec603d9b5c55a6f576'),
    ('build-tools_r36_linux.zip', 'b0b6376977657e8ad9b969bacf4093601da2c6fb'),
)

def run(args, **kwargs):
    subprocess.run([str(value) for value in args], check=True, **kwargs)

def toolchain(cache):
    cache.mkdir(parents=True, exist_ok=True)
    for name, digest in ARTIFACTS:
        archive = cache / name
        if not archive.exists():
            print('Downloading official Android toolchain:', name, flush=True)
            urllib.request.urlretrieve('https://dl.google.com/android/repository/' + name, archive)
        if hashlib.sha1(archive.read_bytes()).hexdigest() != digest:
            raise SystemExit('Android toolchain checksum mismatch: ' + name)
        with zipfile.ZipFile(archive) as package:
            for info in package.infolist():
                destination = (cache / info.filename).resolve()
                if not destination.is_relative_to(cache.resolve()):
                    raise SystemExit('Invalid toolchain archive path')
                if destination.exists():
                    continue
                package.extract(info, cache)
                mode = (info.external_attr >> 16) & 0o777
                if mode and destination.is_file():
                    destination.chmod(mode)
    android = next(cache.rglob('android.jar'))
    aapt = next(cache.rglob('aapt2'))
    return android, aapt.parent

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--assets', type=Path, default=PROJECT/'docs/encrypt')
    parser.add_argument('--output', type=Path, default=ROOT/'build'/f'CipherEnc-{VERSION}-Android-evaluation.apk')
    parser.add_argument('--cache', type=Path, default=Path.home()/'.cache/cipherenc-android-toolchain')
    parser.add_argument('--offline', action='store_true', help='require the verified toolchain archives already in cache')
    args = parser.parse_args()
    if not (args.assets/'index.html').is_file():
        raise SystemExit('Build the browser assets first: missing ' + str(args.assets/'index.html'))
    if args.offline and any(not (args.cache/name).is_file() for name, _ in ARTIFACTS):
        raise SystemExit('Offline build requested but Android toolchain archive is missing')
    android, tools = toolchain(args.cache.resolve())
    build = ROOT/'build'
    if build.exists(): shutil.rmtree(build)
    for name in ['assets/web','classes','dex']: (build/name).mkdir(parents=True)
    shutil.copytree(args.assets, build/'assets/web', dirs_exist_ok=True)
    shutil.copy2(ROOT/'android-bridge.js', build/'assets/web/android-bridge.js')
    index = build/'assets/web/index.html'
    index.write_text(index.read_text().replace('</body>', '<script src="android-bridge.js"></script></body>'))
    with (build/'assets/web/browser.css').open('a') as css:
        css.write('\n.android-note{margin:1rem;padding:1rem;border:1px solid #d8e2ee;background:#f3f7fd;color:#0a1d3a;font:1rem/1.5 system-ui,sans-serif}\n')
    hashes = {str(path.relative_to(build/'assets/web')):hashlib.sha256(path.read_bytes()).hexdigest()
              for path in sorted((build/'assets/web').rglob('*')) if path.is_file() and path.name != 'integrity.json'}
    (build/'assets/web/integrity.json').write_text(json.dumps({'sha256':hashes},indent=2)+'\n')
    run([tools/'aapt2','compile','--dir',ROOT/'res','-o',build/'resources.zip'])
    run([tools/'aapt2','link','-I',android,'--manifest',ROOT/'AndroidManifest.xml','--min-sdk-version','26','--target-sdk-version','36','-o',build/'unsigned.apk','-A',build/'assets',build/'resources.zip'])
    sources = sorted((ROOT/'src').rglob('*.java'))
    run(['java','com.sun.tools.javac.Main','-source','8','-target','8','-bootclasspath',str(tools/'core-lambda-stubs.jar')+os.pathsep+str(android),'-encoding','UTF-8','-d',build/'classes',*sources])
    classes = sorted((build/'classes').rglob('*.class'))
    run(['java','-cp',tools/'lib/d8.jar','com.android.tools.r8.D8','--release','--min-api','26','--lib',android,'--output',build/'dex',*classes])
    with zipfile.ZipFile(build/'unsigned.apk','a',zipfile.ZIP_DEFLATED) as apk:
        for dex in (build/'dex').glob('*.dex'): apk.write(dex,dex.name)
    run([tools/'zipalign','-f','4',build/'unsigned.apk',build/'aligned.apk'])

    # This is an evaluation identity, not the owner's future production publisher key.
    signing = Path.home()/'.local/share/cipherenc/android-evaluation-signing'
    signing.mkdir(parents=True,exist_ok=True,mode=0o700)
    signing.chmod(0o700)
    password_file = signing/'password'
    keystore = signing/'evaluation.p12'
    if not password_file.exists():
        password_file.write_text(secrets.token_urlsafe(32)+'\n')
        password_file.chmod(0o600)
    env = {**os.environ,'CIPHERENC_EVALUATION_KEY_PASSWORD':password_file.read_text().strip()}
    if not keystore.exists():
        run(['keytool','-genkeypair','-keystore',keystore,'-storetype','PKCS12','-alias','cipherenc-evaluation','-keyalg','RSA','-keysize','3072','-validity','10000','-dname','CN=CipherEnc Evaluation, OU=Development Evaluation, O=CipherEnc','-storepass:env','CIPHERENC_EVALUATION_KEY_PASSWORD'],env=env)
        keystore.chmod(0o600)
    args.output.parent.mkdir(parents=True,exist_ok=True)
    run(['java','-jar',tools/'lib/apksigner.jar','sign','--ks',keystore,'--ks-key-alias','cipherenc-evaluation','--ks-pass','env:CIPHERENC_EVALUATION_KEY_PASSWORD','--v4-signing-enabled','false','--out',args.output,build/'aligned.apk'],env=env)
    run(['java','-jar',tools/'lib/apksigner.jar','verify','--verbose','--print-certs',args.output])
    run([tools/'aapt2','dump','permissions',args.output])
    report = {'artifact':args.output.name,'sha256':hashlib.sha256(args.output.read_bytes()).hexdigest(),'signing':'development evaluation identity','minimum_android_api':26,'target_android_api':36,'internet_permission':False,'runtime':'Android system WebView; packaged local CipherEnc browser assets','physical_device_test':'NOT RUN','emulator_test':'NOT RUN'}
    (args.output.parent/'Android-build-verification.json').write_text(json.dumps(report,indent=2)+'\n')
    shutil.copy2(ROOT/'README.md',args.output.parent/'README.md')
    print('Built Android evaluation APK:',args.output)

if __name__ == '__main__': main()
