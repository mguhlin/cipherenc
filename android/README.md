# CipherEnc Android evaluation app

This is an installable Android APK containing the CipherEnc browser interface and all of its cryptographic assets. It uses Android's built-in WebView inside its own application window; it does not launch an external browser. It is **not the native Go/Tk desktop application**, and does not use native Android widgets for the encryption workspace.

The APK is **development-signed for evaluation**. It has not been exercised on physical Android hardware or an emulator. Do not describe it as production-ready, Play Store approved, independently audited, or device-tested. Android 8.0/API 26 or later and a modern Android system WebView (Chromium 100 or later) are required. Actual WebView compatibility remains unverified until device testing.

## Offline behavior and limits

- HTML, JavaScript, CSS, Argon2id, OpenPGP, and artwork assets are bundled in the APK.
- There is no `INTERNET` permission, remote updater, analytics, cloud account, or runtime asset download.
- Only the exact packaged application origin is served. Other resource requests and external navigation are denied.
- The JavaScript bridge is exposed only to the packaged local interface. It provides bounded Android document-picker operations, not arbitrary filesystem access or a general command runner.
- Files and folders to encrypt are limited to 16 MiB total and 1,000 files. Encrypted input and saved packages are limited to 24 MiB. Use the browser or desktop edition for larger work.
- Folder selection and restoration use Android's Storage Access Framework. Restored folders are placed inside a new randomly named `CipherEnc-restored-...` folder; existing files are not replaced.
- Saved files go to the provider/location chosen in Android's picker. A separately selected cloud-storage provider may synchronize its own files; this app's lack of network permission does not disable other apps or the operating system.
- Input and output are held in memory until cleared. Downloads, clipboard contents, OS copies, document-provider copies and backups remain under the user's control.

## Build

The build script uses official Google Android SDK platform 36 and build-tools 36.0.0 archives, verified against the SHA-1 checksums in Google's repository manifest. These are tool-distribution checksums, not an independent security audit. The first build downloads the archives into `~/.cache/cipherenc-android-toolchain/`; later builds can run offline. Java 21 with the `jdk.compiler` module is required. No Gradle or NDK is needed.

From the repository root:

```sh
python3 android/build.py --assets docs/encrypt
```

With toolchain archives already cached:

```sh
python3 android/build.py --offline --assets docs/encrypt
```

The script compiles Java, creates DEX bytecode, packages local web assets, aligns the APK, signs it, verifies its signing schemes, and dumps declared permissions. `Android-build-verification.json` records the artifact hash and actual test limits without a maintenance timestamp.

The default output is `android/build/CipherEnc-Android-evaluation.apk`; the release packager can copy it into the organized distribution folder. A local network-namespace build with no internet route completed successfully. `apksigner verify` confirmed APK Signature Scheme v2 and v3 signatures, and `aapt2 dump permissions` returned only `package: org.mguhlin.cipherenc`, with no declared permissions. These build checks do not establish device behavior.

The dedicated evaluation keystore and its password are stored outside the repository under `~/.local/share/cipherenc/android-evaluation-signing/`, with private permissions. They are excluded from the project, source archives and build outputs. Keep them private if evaluation updates need to retain the same signing identity. A production Android distribution needs an owner-managed signing identity, device/emulator workflow tests, screen-reader and touch checks, lifecycle/recovery testing, and update/release procedures before promotion.

## Development checks

`android/tests/PathValidationTest.java` exercises the actual compiled bridge path validator on the JVM against ordinary names, traversal, absolute paths, backslashes, colons, and control characters. `node android/test-bridge.cjs` exercises the JavaScript adapter in desktop Chromium: a late folder callback cannot restore cleared work, size checks precede the browser crypto handler, and saving passes the intended filename and bytes to the bounded bridge. Both checks passed locally. They do not emulate Android's WebView, document providers, picker UI, lifecycle, or permission enforcement; those remain device/emulator checks.

## Source references

- [Android: Load in-app content](https://developer.android.com/develop/ui/views/layout/webapps/load-local-content)
- [Android: Storage Access Framework](https://developer.android.com/training/data-storage/shared/documents-files)
- [Android: D8 compiler](https://developer.android.com/tools/d8)
- [Android: APK signing tools](https://developer.android.com/tools/apksigner)

The CipherEnc application remains AGPL-3.0-or-later. Browser library licenses remain in the packaged web assets and repository notices. Android SDK tooling and platform files retain their own Google/Android terms and are not redistributed in the APK source ZIP.
