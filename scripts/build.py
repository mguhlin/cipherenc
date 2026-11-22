#!/usr/bin/env python3
"""Build offline native executables and an organized local/download distribution."""
import argparse
import hashlib
import os
from pathlib import Path
import plistlib
import shutil
import subprocess
import zipfile

root=Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser();parser.add_argument('--output',type=Path,default=root/'dist');parser.add_argument('--package-only',action='store_true');args=parser.parse_args()
out=args.output.resolve();out.mkdir(parents=True,exist_ok=True)
support=out/'Support';downloads=support/'Downloads';licenses=support/'Licenses'
downloads.mkdir(parents=True,exist_ok=True);licenses.mkdir(parents=True,exist_ok=True)
docs=['README.md','VCC-REVIEW.md']
notices=['LICENSE','THIRD-PARTY-NOTICES.md','UPSTREAM-LICENSE','UPSTREAM-NOTICE.md','GO-LICENSE']
for name in docs:shutil.copy2(root/name,support/name)
for name in notices:shutil.copy2(root/name,licenses/name)
shutil.copy2(root/'internal/tkdnd/LICENSE',licenses/'TkDND-LICENSE')
# Preserve all third-party licenses without putting developer material in the launch folders.
for folder in (root/'vendor').rglob('*'):
 if folder.is_file() and ('license' in folder.name.lower() or folder.name in ['COPYING','AUTHORS','NOTICE']):
  target=licenses/'Go-dependencies'/folder.relative_to(root/'vendor');target.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(folder,target)
targets=[('windows','amd64','Windows','x64','Windows-x64'),('windows','arm64','Windows','ARM64','Windows-ARM64'),('darwin','amd64','Mac','Intel','macOS-Intel'),('darwin','arm64','Mac','Apple Silicon','macOS-Apple-Silicon'),('linux','amd64','Linux','x64','Linux-x64'),('linux','arm64','Linux','ARM64','Linux-ARM64')]
archives=[]
for goos,arch,platform,kind,label in targets:
 folder=out/platform/kind;folder.mkdir(parents=True,exist_ok=True)
 executable=folder/('CipherEnc.exe' if goos=='windows' else 'CipherEnc')
 if goos=='darwin':
  contents=folder/'CipherEnc.app'/'Contents';(contents/'MacOS').mkdir(parents=True,exist_ok=True);(contents/'Resources').mkdir(exist_ok=True)
  executable=contents/'MacOS'/'CipherEnc'
  info={'CFBundleName':'CipherEnc','CFBundleDisplayName':'CipherEnc','CFBundleIdentifier':'org.mguhlin.cipherenc','CFBundleVersion':'1.1.1','CFBundleShortVersionString':'1.1.1','CFBundleExecutable':'CipherEnc','CFBundlePackageType':'APPL','NSHighResolutionCapable':True,'NSHumanReadableCopyright':'CipherEnc contributors. AGPL-3.0-or-later; see bundled notices.'}
  (contents/'Info.plist').write_bytes(plistlib.dumps(info));(contents/'PkgInfo').write_bytes(b'APPL????')
  for name in docs+notices:shutil.copy2(root/name,contents/'Resources'/name)
 if not args.package_only:
  env={**os.environ,'GOOS':goos,'GOARCH':arch,'CGO_ENABLED':'0','GOPROXY':'off','GOSUMDB':'off','GOTOOLCHAIN':'local'}
  flags='-s -w -buildid=' + (' -H windowsgui' if goos=='windows' else '')
  subprocess.run(['go','build','-mod=vendor','-trimpath','-buildvcs=false','-ldflags',flags,'-o',str(executable),'.'],cwd=root,env=env,check=True)
 if not executable.is_file():raise SystemExit('Missing native executable: '+str(executable))
 executable.chmod(0o755)
 archive=downloads/f'CipherEnc-1.1.1-{label}.zip'
 with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED,compresslevel=6) as z:
  package=Path(f'CipherEnc-{label}')
  for p in sorted(folder.rglob('*')):
   if p.is_file():z.write(p,package/p.relative_to(folder))
  for name in docs:z.write(root/name,package/'Support'/name)
  for name in notices:z.write(root/name,package/'Support'/'Licenses'/name)
  z.write(root/'internal/tkdnd/LICENSE',package/'Support'/'Licenses'/'TkDND-LICENSE')
  z.writestr(str(package/'START HERE.txt'),'CipherEnc native desktop\n\nRun CipherEnc.exe on Windows, CipherEnc.app on Mac, or CipherEnc on Linux.\nNo browser or internet connection is required.\nDrop files/folders, or enter text. The action is detected automatically.\nFiles/folders are zipped in memory, encrypted and verified beside their originals.\nEncrypted files use .cen. Existing files are never replaced.\nShredding is optional, asks after verification, and is best effort.\nSee Support/README.md for platform and format details.\n')
 archives.append(archive);print('Built native',label,flush=True)

guides={
 'Windows':'Most Windows computers: open x64, then run CipherEnc.exe.\nFor a Windows-on-ARM device: open ARM64 instead.\nThe app opens its own native window and needs no browser.\nThis build has no publisher certificate; your computer may require app-specific approval.\n',
 'Mac':'Apple M-series Mac: open Apple Silicon, then CipherEnc.app.\nIntel Mac: open Intel, then CipherEnc.app.\nThe app opens a native Mac window; no browser or XQuartz is needed.\nThis build is not Apple notarized; use the app-specific OS review process.\n',
 'Linux':'Most Linux desktops: open x64, then run CipherEnc.\nARM64 Linux: use ARM64/CipherEnc instead.\nYou can also use Start CipherEnc.desktop on this x64 computer.\nA normal graphical Linux desktop with X11/XWayland libraries is required.\nNo browser or internet connection is required.\n'
}
for platform,text in guides.items():(out/platform/'START HERE.txt').write_text('CipherEnc — '+platform+'\n\n'+text+'\nRead ../Support/README.md for help.\n')
shutil.copy2(root/'docs/encrypt/assets/cipherenc-mark.svg',support/'CipherEnc.svg')
launcher=out/'Linux'/'Start CipherEnc.desktop'
launcher.write_text(f'[Desktop Entry]\nType=Application\nName=CipherEnc\nComment=Native offline encryption\nExec="{out}/Linux/x64/CipherEnc"\nIcon={support}/CipherEnc.svg\nTerminal=false\nCategories=Utility;Security;\n');launcher.chmod(0o755)

apk=root/'android/build/CipherEnc-Android-evaluation.apk'
if apk.is_file():
 (out/'Android').mkdir(exist_ok=True);shutil.copy2(apk,out/'Android'/'CipherEnc.apk');copy=downloads/apk.name;shutil.copy2(apk,copy);archives.append(copy)
 (out/'Android'/'START HERE.txt').write_text('CipherEnc Android evaluation APK\n\nInstall CipherEnc.apk only if your device policy permits sideloading.\nThis development-signed Android app bundles the browser tool inside the OS WebView.\nIt uses no external browser and requests no INTERNET permission.\nIt is not the native Go desktop app. Physical Android-device testing remains pending.\n')
source=downloads/'CipherEnc-1.1.1-source.zip'
with zipfile.ZipFile(source,'w',zipfile.ZIP_DEFLATED,compresslevel=6) as z:
 for p in sorted(root.rglob('*')):
  if not p.is_file():continue
  rel=p.relative_to(root)
  if set(rel.parts)&{'.git','node_modules','dist','build','test-results','playwright-report','__pycache__'} or p.suffix in ['.keystore','.jks']:continue
  if p.is_relative_to(out):continue
  z.write(p,Path('CipherEnc-source')/rel)
archives.append(source)
(downloads/'SHA256SUMS.txt').write_text(''.join(f'{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}\n' for p in archives))
(out/'START HERE.txt').write_text('CIPHERENC — START HERE\n\nWindows: open Windows, then START HERE.txt.\nMac: open Mac, then START HERE.txt.\nLinux: open Linux, then Start CipherEnc.desktop or the matching executable.\nAndroid: if present, open Android and read its evaluation-build instructions.\n\nNative desktop encryption works fully offline and needs no browser.\nAll encrypted files end in .cen. Keep your password safe; test decryption before deleting originals.\n\nSupport contains the README, review notes, licenses, checksums, source, and download archives.\nWebsite: https://mguhlin.github.io/cipherenc/\n')
print('Organized distribution:',out,flush=True)
