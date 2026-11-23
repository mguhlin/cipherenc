# CipherEnc website

- `index.html` introduces CipherEnc, lists downloads and answers common questions.
- `encrypt/index.html` is the password workspace for files, folders and text.
- `encrypt/openpgp.html` is the key workspace, generated from the shared interface by `scripts/openpgp_page.py`.
- `encrypt/help.html` contains browser and OpenPGP FAQs. Workspace links open it separately to preserve current input.
- `scripts/templates/openpgp-controls.html` supplies the key controls for the dedicated page and the Android bundle.
- `assets/` contains the banner, three-panel comic and native-app screenshots.

## Preview locally

From the repository root:

```sh
python3 -m http.server 8941 --bind 127.0.0.1 --directory docs
```

Open `http://127.0.0.1:8941/`. The site uses local CSS, system fonts and bundled encryption libraries. Browser cryptography and clipboard features need HTTPS or localhost.

## Update content and versions

- Keep help focused on CipherEnc tasks, using friendly FAQs and short bullet lists.
- Show the release version on the website, workspace, app and download folders.
- `main.go` supplies the version used by the native and Android packagers. Keep browser, website and Android labels consistent with it.
- Run `python3 scripts/update-integrity.py` after changing browser files; it refreshes the generated key page, hashes and asset URLs.
- Run `python3 scripts/check-version.py`, `python3 scripts/update-integrity.py --check` and `npm test` before publishing.
- Keep required dependency licenses with the bundled libraries.
- Avoid maintenance timestamps in public content.

## Files and mobile browsers

- Password-mode files/folders are ZIP-compressed before encryption and automatically unzipped on restoration.
- Single restored files download directly. Folder restoration uses a new directory where supported; other browsers offer individual saves or an optional ZIP.
- Folder picking, empty folders, available memory and save locations depend on the browser and device.
- Native XChaCha20 packages require the desktop app. OpenPGP uses the key workspace.
- Android and iPhone viewport tests do not replace testing on physical devices. The Android APK remains an evaluation build.
