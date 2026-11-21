# CipherEnc website

`index.html` is the download and introduction page. `encrypt/` is the browser-only encryption workspace; it runs independently of the native Go application. `assets/` contains the generated hero/comic artwork and actual Linux native-app screenshots.

The site uses local CSS, system fonts, and locally bundled cryptographic libraries. JSZip 3.10.1 (MIT option) supplies the mobile folder-restoration download; its package metadata and full dual-license notice are under `encrypt/vendor/jszip/`. It requires no package installation or build step to serve. From the repository root, a local preview can be started with:

```sh
python3 -m http.server 8941 --bind 127.0.0.1 --directory docs
```

Open `http://127.0.0.1:8941/`. On production, HTTPS supplies the secure context required by browser cryptography and clipboard features. The meta Content Security Policy blocks application connections in the browser workspace. GitHub Pages response headers and browser/OS activity are outside that meta policy; no response-level framing-policy claim is made.

## Browser workspace

The workspace adapts the imported engine snapshot recorded in the repository's `upstream.json`, retaining its AES-GCM/Argon2id formats, legacy readers, file/folder/text workflow, optional OpenPGP keys, password generator, and nonpersistent language control. Changes include CipherEnc branding, `.cen` download suffixes, local navigation, readable helper/control colors, bounded UI operation state, explicit clear, stale-directory-import invalidation, and save/clipboard failure messages. No API endpoint, content upload, telemetry, or native XChaCha20 bridge is included.

`encrypt/integrity.json` records SHA-256 hashes of shipped files other than the hash document itself. Update those hashes after changing a browser asset. If `app.js` changes, update its SHA-384 `integrity` value in `encrypt/index.html` too. Do not change the upstream crypto bundles without reviewing their licenses, versions, and compatibility.

The `.cen` filename ending does not distinguish AES, OpenPGP, or native-app XChaCha20 packages. The browser reads supported password formats; native XChaCha20 packages require the native desktop app.

## Mobile limits

File and text workflows are designed for Android and iOS browsers. Folder picking depends on the OS picker; direct directory restoration is not available in every browser. The fallback downloads one unencrypted ZIP preserving safe folder paths. Duplicate normalized paths and unsafe portable filenames are rejected before archive creation. Empty directories are not preserved. Phones may suspend a background tab, and real package sizes are constrained by available memory, free storage, and download behavior.

Desktop Chromium/Firefox/WebKit and emulated Pixel 5/iPhone 13 browser profiles are development test configurations. They do not establish testing on physical Android/iOS hardware. The Android APK is a separate evaluation container using the system WebView and bundled workspace; see the Android build notes and its stated test limits.

## Design and review

The page follows VCC's function-first approach: show the browser action and platform downloads early, explain the reversible password/package workflow with an accessible three-panel comic, show actual app captures, and separate format facts from certification claims. The palette uses blue, silver, and navy. Web design, SecureVCC, education privacy, and Rescue guidance inform the implementation; these are engineering checks, not security/accessibility/legal certifications.

Avoid visitor-facing maintenance timestamps in source, templates, generated metadata, and downloads. Image captions describe what was captured, not when it was captured. Keep generated-image prompts and provenance with project development notes rather than embedding a date in the public page.
