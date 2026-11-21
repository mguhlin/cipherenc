# CipherEnc

CipherEnc protects files, folders, and text on your own device. Encrypted packages use **`.cen`**. Its interface uses a blue, silver, and navy palette.

- **Native desktop app:** Windows, Mac, and Linux. A Go application with its own native window, AES-256-GCM and XChaCha20-Poly1305. No browser, internet connection, account, installer, or runtime download is required.
- **Browser tool:** [Open CipherEnc](https://mguhlin.github.io/cipherenc/encrypt/). Files, folders, and text are processed in browser memory using locally hosted cryptographic assets. Works with modern desktop, Android and iOS browsers, subject to their file-picker and memory limits.
- **Android evaluation APK:** an installable app bundling the browser tool in Android's system WebView. No external browser and no INTERNET permission. This is a development-signed evaluation build, not the Go desktop application; device/emulator testing remains pending.

Website and downloads: https://mguhlin.github.io/cipherenc/

## Start here

Open `dist/START HERE.txt`, then choose **Windows**, **Mac**, or **Linux**. The Desktop `CipherEnc` shortcut points to this organized distribution; the actual files live in this repository's `dist/` folder. Root distribution folders contain the entry points. Manuals, licenses, checksums, source and ZIP downloads are under `Support/`.

| Device | Native application |
| --- | --- |
| Windows with Intel/AMD processor | `Windows/x64/CipherEnc.exe` |
| Windows on ARM | `Windows/ARM64/CipherEnc.exe` |
| Mac with Apple Silicon | `Mac/Apple Silicon/CipherEnc.app` |
| Mac with Intel processor | `Mac/Intel/CipherEnc.app` |
| Linux x64 desktop | `Linux/x64/CipherEnc` or `Linux/Start CipherEnc.desktop` |
| Linux ARM64 desktop | `Linux/ARM64/CipherEnc` |

Extract a downloaded ZIP before launching. Linux needs a normal graphical desktop with X11/XWayland and its standard X11/font libraries. Windows and Mac use their native window systems; Mac does not need XQuartz. The bundled GUI runtime is extracted to a private user cache. It contains application libraries, not documents, passwords or saved work.

The desktop builds do not have Microsoft publisher certificates or Apple Developer ID notarization. An unfamiliar application may need app-specific OS approval; managed devices may require IT approval. Do not disable system-wide security settings. [Apple's guidance](https://support.apple.com/en-us/102445) describes its review process.

## Encrypt and restore

1. Choose **Encrypt**, then File, Folder, or Text. Select your data or enter a note.
2. Keep AES-256-GCM selected, or choose XChaCha20-Poly1305 for one file/note. Enter and confirm a strong password; the generator can create one.
3. Run the action and explicitly save the `.cen` result. The native app never replaces an existing file.
4. Test it: choose **Decrypt**, select the saved package, enter its password, run, and save the restored result. Keep the original until you have checked the restored copy.

Passwords cannot be recovered. A `.cen` extension is only a filename ending; authenticated contents determine the format. The password derives a cryptographic key; the comic's physical key is a visual metaphor.

**Verify** authenticates a package and discards restored plaintext. It detects alterations and incorrect passwords; it does not identify the sender.

**AES-256-GCM** is the default. The native app uses the CS4 container with Argon2id v1.3, 64 MiB, three passes, one lane, a fresh 128-bit salt, independently authenticated chunks, and encrypted file metadata. It supports up to 10,000 files totaling 256 MiB per package. Empty folders are not recorded. It also reads legacy CS2/CS3 and text v1/v2/v3 containers, including saved PRISMVAULT notes. Native and browser CS4 interoperability is tested in both directions.

**XChaCha20-Poly1305** uses a 256-bit key and a random 192-bit nonce with fixed-cost Argon2id. It supports one file or note up to 16 MiB. Its `.cen` packages use CipherEnc's `CENCX01!` format, which is desktop-only and is not understood by the browser tool or Paranoia. The header is authenticated; filename and contents are encrypted. This new container has not had an independent cryptographic audit.

**OpenPGP** is available in the browser and Android editions through the pinned local OpenPGP.js library. The native Go desktop edition does not include OpenPGP. Check recipient fingerprints independently; integrity verification is not signature verification.

RC6, Serpent, Blowfish, Twofish, Threefish, SHACAL-2, and custom cascades are not included. Larger advertised key totals do not establish stronger protection. The Go project's [Twofish documentation](https://pkg.go.dev/golang.org/x/crypto/twofish) recommends AES or XChaCha20-Poly1305 for new applications. No equivalence or compatibility with Paranoia C4 is claimed.

## Mobile browser details

The browser tool supports text and file encryption/decryption on modern Android/iOS browsers. Folder selection depends on what the system picker exposes. Chromium browsers can restore directly to a directory where that API is available; other browsers receive one ZIP preserving the restored tree. The ZIP contains plaintext, so save and share it deliberately. Mobile memory, browser suspension, and OS download behavior impose practical limits. Keep the tool foregrounded during an operation.

Desktop browser engines and Android/iPhone viewport profiles are tested on Linux. Physical Android/iOS devices and VoiceOver/TalkBack are not tested here. The Android APK has separate conservative size limits and lifecycle limitations; see [android/README.md](android/README.md).

## Privacy and safety

The native app has no local web server, browser dependency, telemetry, update service, network client, account, or database. User-selected data and keys are processed in memory. Nothing is automatically saved. Save creates a new file; folder restoration creates a new randomly named directory and confines paths with Go's `os.Root` APIs. Originals and existing destination files are not overwritten.

The browser edition serves code/assets through GitHub Pages; the host receives ordinary page requests. Encryption inputs, filenames, passwords and keys are not uploaded by the tool. It uses no app cookies, localStorage, sessionStorage, IndexedDB, remote fonts or analytics. Locally bundled scripts and CSP restrict resource loading. Language selection is not persisted.

Clear workspace releases active inputs and result references. Native idle clearing runs after ten minutes without interaction. Passwords clear after a successful result. Downloads, clipboard contents, screenshots, document-provider copies, OS swap, backups, extensions, and managed-runtime memory reclamation remain outside app erasure guarantees. A compromised device can expose data. The Android app's no-network permission does not prevent a separately chosen cloud document provider from synchronizing its own files.

School use requires the institution's own review for intended ages, data and device policies. No FERPA, COPPA, accessibility, cryptographic, or school-approval certification is claimed.

## Source layout and reproducible checks

| Path | Responsibility |
| --- | --- |
| `main.go`, `gui.go` | Native launch, controls, operation lifecycle and deliberate save actions |
| `vault.go`, `crypto.go` | CS4-compatible AES containers and the separate XChaCha20 container |
| `vault_test.go`, `selftest.go` | Synthetic round trips, limits, denied cases, restoration and executable self-check |
| `docs/` | GitHub Pages site, real screenshots, artwork, comic and browser tool |
| `android/` | Offline Android container, SDK build script and evaluation limitations |
| `scripts/` | Native builds, compatibility fixtures and validation helpers |
| `reference/` | Imported engine references and independent compatibility fixtures |
| `dist/` | Organized executable distribution; excluded from Git |
| `vendor/` | Local offline build dependencies; excluded from Git and included in the release source ZIP |

Go dependencies are pinned by `go.mod` and `go.sum`. A fresh Git checkout needs a matching Go toolchain and one initial dependency download. The full release source ZIP includes vendored dependencies for offline rebuilds:

```sh
go mod download
go mod vendor
go test -mod=vendor -race ./...
go vet -mod=vendor ./...
python3 scripts/build.py
```

The build script disables network module fetching and cross-compiles all six desktop targets with `CGO_ENABLED=0`. Run a native executable's synthetic self-check with `CipherEnc --self-test /path/to/empty-scratch-folder`. `--ui-smoke` is a maintainer test that opens a window, exercises a synthetic note and exits; `--version` prints the build version. Passwords are never command-line arguments.

Browser development checks use pinned Playwright/axe-core dependencies: `npm ci --ignore-scripts`, install the matching Playwright browsers, then run the documented scripts. `scripts/browser-interop.cjs` verifies native CS4 output in the independent browser engine and generates the committed browser-to-Go fixtures. The optional `scripts/independent_fixture.py` uses system libargon2/libsodium solely to reproduce a test fixture, not as app runtime dependencies.

`dist/Support/Downloads/SHA256SUMS.txt` covers distribution archives. See [VCC-REVIEW.md](VCC-REVIEW.md) for actual checks and unverified boundaries. Cross-compilation is not physical-device testing.

## License and provenance

CipherEnc and its incorporated engine code are AGPL-3.0-or-later. See `LICENSE`, `UPSTREAM-LICENSE`, `UPSTREAM-NOTICE.md`, `THIRD-PARTY-NOTICES.md`, and `upstream.json`. Keep third-party license notices with redistribution. Corresponding application source, browser dependencies, and vendored Go dependency source are supplied in the release source archive. User files, notes, keys and outputs remain the user's property.
