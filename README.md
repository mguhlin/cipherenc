# CipherEnc

CipherEnc protects files, folders, and text on your own device. Encrypted packages use **`.cen`**. Its interface uses a blue, silver, and navy palette.

- **Native desktop app:** Windows, Mac, and Linux. A Go application with its own native window, AES-256-GCM and XChaCha20-Poly1305. No browser, internet connection, account, installer, or runtime download is required. Files/folders are zipped, encrypted and verified beside their originals; optional shredding requires a post-verification choice.
- **Browser tool:** [Open CipherEnc](https://mguhlin.github.io/cipherenc/encrypt/). Files, folders, and text are processed in browser memory using locally hosted cryptographic assets. Works with modern desktop, Android and iOS browsers, subject to their file-picker and memory limits.
- **Android evaluation APK:** an installable app bundling the browser tool in Android's system WebView. No external browser and no INTERNET permission. This is a development-signed evaluation build, not the Go desktop application; device/emulator testing remains pending.

**Current version: 1.1.2.** The version appears in the app window, website and extracted download-folder names. Install updates by downloading the current release and reopening the refreshed app.

Website and downloads: https://mguhlin.github.io/cipherenc/

## Start here

Open `dist/CipherEnc-1.1.2/START HERE.txt`, then choose **Windows**, **Mac**, or **Linux**. The Desktop `CipherEnc-1.1.2` folder and `CipherEnc` shortcut point to this organized distribution; the actual files live in this repository's `dist/CipherEnc-1.1.2/` folder. Root distribution folders contain the entry points. Manuals, licenses, checksums, source and ZIP downloads are under `Support/`.

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

## Drop files or paste text

There are no action, file-vault, text-vault, or integrity tabs. Add original files/folders to encrypt, or a recognized encrypted package to decrypt. CipherEnc inspects the contents; renaming a file does not change its format. A `.cen` file with unrecognized contents is rejected instead of silently encrypted again.

**Desktop:** enter and confirm a strong password, then drop files/folders into the box or use **Choose files / Choose folder**. Each top-level item is ZIP-compressed in memory, encrypted with AES-256-GCM, saved beside its original as `name.cen`, and reopened for authenticated verification. Existing outputs receive a numbered name. If you choose inputs first, enter and confirm the password; matching fields start encryption automatically after a short typing/paste pause. There is no need to click **Process selected** for files or folders. A mismatch waits, and Clear cancels a pending start. If you generate a password after selecting originals, copy/save it and paste it into the confirmation field to start. For encrypted inputs selected first, enter the password and press Enter to finish password entry. Dropping an encrypted package restores into a new folder beside it after authentication. ZIP extraction rejects unsafe paths, links, duplicate portable names and excessive expansion; empty directories from desktop selections are preserved.

Type or paste text in the same window. CipherEnc detects supported encrypted text automatically. Press **Process selected**: encrypted notes are saved through the system save picker, while decrypted text appears in the text area. AES notes use the pasteable PRISMVAULT3 format shared with the browser. **Advanced options** holds the optional desktop-only XChaCha20 note algorithm. A retained result can be saved again if a save was cancelled.

**Browser:** use the same drop box and text area, with automatic action detection and integrity checks. Password-mode files/folders are ZIP-compressed in memory, encrypted as `.cen`, and independently decrypted and compared before saving. With credentials ready, choosing/dropping inputs starts automatically. Otherwise enter the password and press the single action button. Text encryption displays a copyable encrypted text box and produces a `.cen` download; decrypted text appears in that same result area. Numbered instructions guide password, content and result steps.

Encrypted results save automatically. Supporting desktop browsers ask for a location when the action starts, so the browser's required user gesture is preserved; encryption and verification finish before the write. Other browsers automatically trigger their normal download. Their own settings determine whether to ask for a destination. Cancelling the save picker keeps the input ready. A failed write retains the result for retry. The browser cannot choose the original file's directory automatically. Browser decryption automatically unzips CipherEnc archives. A single original file downloads directly, rather than leaving a ZIP. Supporting browsers ask where to restore and write into a new restoration folder; existing files are not replaced. Browsers without folder-writing access show individual unzipped files to save. A ZIP is an explicit alternative for those browsers, rather than the default decrypted result.

Passwords cannot be recovered. Integrity authentication detects incorrect passwords and alterations; it does not identify a sender. OpenPGP encryption with a recipient's public key cannot be decrypted for a round-trip check without their private key; OpenPGP decryption performs the library's integrity checks.

## Optional original shredding

The **desktop app asks after each encrypted archive is saved and verified**. **No / Keep original is the default**. Choosing Yes authorizes one random overwrite of accessible file bytes, a filesystem sync, and removal of the original file or folder. The app rechecks the saved archive and the originals immediately before shredding. Changed selections, symlinks and hard-linked files are refused. Directories are removed only when empty; source trees are never recursively deleted with `RemoveAll`. Close editors and other processes that can change the originals first.

This is **best effort**, not guaranteed physical erasure: SSD controllers, copy-on-write filesystems, journals, snapshots, cloud synchronization and backups may retain copies. A failure can leave some originals present or partly overwritten; the verified archive is retained. Only choose shredding if you have recorded the password and accept that the original cannot be recovered through CipherEnc. See [GNU's explanation of overwrite limitations](https://www.gnu.org/software/coreutils/manual/html_node/shred-invocation.html).

**The browser and Android editions do not shred originals.** Browser-selected files are generally read-only copies, and browser APIs cannot reliably overwrite the underlying storage. Keep or remove originals using the device's own file-management tools and retention policy.

**AES-256-GCM** is the default. The native app uses the CS4 container with Argon2id v1.3, 64 MiB, three passes, one lane, a fresh 128-bit salt, independently authenticated chunks, and encrypted file metadata. It supports up to 10,000 files totaling 256 MiB per package. The desktop ZIP workflow preserves empty directories; browser folder pickers generally expose only files. It also reads legacy CS2/CS3 and text v1/v2/v3 containers, including saved PRISMVAULT notes. Native and browser CS4 interoperability is tested in both directions.

**XChaCha20-Poly1305** uses a 256-bit key and a random 192-bit nonce with fixed-cost Argon2id. It supports one file or note up to 16 MiB. Its `.cen` packages use CipherEnc's `CENCX01!` format, which is desktop-only and requires the native app. The header is authenticated; filename and contents are encrypted. This new container has not had an independent cryptographic audit.

**OpenPGP** is available in the browser and Android editions through the pinned local OpenPGP.js library. The native Go desktop edition does not include OpenPGP. Check recipient fingerprints independently; integrity verification is not signature verification.

The browser’s **Use OpenPGP** link opens a [dedicated key workspace](https://mguhlin.github.io/cipherenc/encrypt/openpgp.html). Paste a complete public/private key block in Step 1, or import an exported armored `.asc` or binary `.pgp` key. Its fingerprint appears locally. Encrypt with the recipient’s public key; decrypt with the matching private key and its passphrase, if protected. Step 2 detects encrypted messages automatically. Text results are copyable, and files retain their original filename on decryption. This workspace handles one file or message; ZIP a folder first, or use password mode for automatic folder packaging.

There is no persistent account or key-server link. Export existing keys from a trusted key manager, then import/paste them for each operation; keys clear after success, Clear or inactivity. Key URLs are not fetched. This tool does not generate key pairs or verify sender signatures. Android’s **Use OpenPGP** link opens these key controls within its bundled offline workspace. The desktop Go apps keep their native password interface and AES/XChaCha20 formats; web interface changes do not add OpenPGP to those executables.


## Mobile browser details

The browser tool supports text and file encryption/decryption on modern Android/iOS browsers. Folder selection depends on what the system picker exposes. Browsers with directory access restore the unzipped tree into a new folder. Other browsers download single restored files directly and offer individual saves for a folder; an optional ZIP can preserve its full tree. Restored contents are plaintext, so save and share deliberately. Mobile memory, browser suspension, and OS download behavior impose practical limits. Keep the tool foregrounded during an operation.

Desktop browser engines and Android/iPhone viewport profiles are tested on Linux. Physical Android/iOS devices and VoiceOver/TalkBack are not tested here. The Android APK has separate conservative size limits and lifecycle limitations; see [android/README.md](android/README.md).

## Privacy and safety

The native app has no local web server, browser dependency, telemetry, update service, network client, account, or database. User-selected data and keys are processed in memory. The desktop file workflow automatically saves a new encrypted archive beside each original and verifies it. Text uses the system save picker. Save creates a new file; folder restoration creates a new randomly named directory and confines paths with Go's `os.Root` APIs. Existing destination files are not overwritten. Originals are retained unless the desktop user explicitly confirms best-effort shredding after verification.

The browser edition serves code/assets through the website; its host receives ordinary page requests. Encryption inputs, filenames, passwords and keys are not uploaded by the tool. It uses no app cookies, localStorage, sessionStorage, IndexedDB, remote fonts or analytics. Locally bundled scripts and CSP restrict resource loading. Language selection is not persisted.

Clear workspace releases active inputs and result references. Native idle clearing runs after ten minutes without interaction. Passwords clear after a successful result. Downloads, clipboard contents, screenshots, document-provider copies, OS swap, backups, extensions, and managed-runtime memory reclamation remain outside app erasure guarantees. A compromised device can expose data. The Android app's no-network permission does not prevent a separately chosen cloud document provider from synchronizing its own files.

School use requires the institution's own review for intended ages, data and device policies. No FERPA, COPPA, accessibility, cryptographic, or school-approval certification is claimed.

## Source layout and reproducible checks

| Path | Responsibility |
| --- | --- |
| `main.go`, `gui.go` | Native launch, unified workspace, operation lifecycle and save/shred consent |
| `archive.go`, `shred_*.go` | In-memory ZIP, adjacent saves, safe restoration and opt-in best-effort shredding |
| `internal/tkdnd/` | Bundled native drag-and-drop libraries, pinned provenance and licenses |
| `vault.go`, `crypto.go` | CS4-compatible AES containers and the separate XChaCha20 container |
| `vault_test.go`, `archive_test.go`, `selftest.go` | Synthetic round trips, limits, denied cases, restoration and executable self-check |
| `docs/` | Website, real screenshots, artwork, comic and browser tool |
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

The build script disables network module fetching and cross-compiles all six desktop targets with `CGO_ENABLED=0`. Run a native executable's synthetic self-check with `CipherEnc --self-test /path/to/empty-scratch-folder`. `--ui-smoke` is a maintainer test that opens a window, exercises a synthetic archive and exits; `--version` prints the build version. Passwords are never command-line arguments.

Browser development checks use pinned Playwright/axe-core dependencies: `npm ci --ignore-scripts`, install the matching Playwright browsers, then run the documented scripts. `scripts/browser-interop.cjs` verifies native CS4 output in the independent browser engine and generates the committed browser-to-Go fixtures. The optional `scripts/independent_fixture.py` uses system libargon2/libsodium solely to reproduce a test fixture, not as app runtime dependencies.

`npm test` also checks the dedicated OpenPGP interface with synthetic keys, armored and binary key import, displayed fingerprints, protected-key retry, exact text/file restoration, key clearing, inaccessible key-server URLs, accessibility and mobile viewport layouts. `scripts/openpgp_page.py` derives that page from the shared workspace; `python3 scripts/update-integrity.py` refreshes it and the pinned asset hashes. Its `--check` option detects stale generated pages. Android packages these assets when rebuilt; it does not fetch website updates at runtime. Existing desktop/Android installations receive changes by downloading and installing the refreshed app.

`dist/CipherEnc-1.1.2/Support/Downloads/SHA256SUMS.txt` covers distribution archives. See [VCC-REVIEW.md](VCC-REVIEW.md) for actual checks and unverified boundaries. Cross-compilation is not physical-device testing.

## License and provenance

CipherEnc and its incorporated engine code are AGPL-3.0-or-later. See `LICENSE`, `UPSTREAM-LICENSE`, `UPSTREAM-NOTICE.md`, `THIRD-PARTY-NOTICES.md`, and `upstream.json`. Keep third-party license notices with redistribution. Corresponding application source, browser dependencies, and vendored Go dependency source are supplied in the release source archive. User files, notes, keys and outputs remain the user's property.

Native drag-and-drop uses bundled Tcl-9-compatible TkDND libraries; their origins and hashes are recorded in `internal/tkdnd/manifest.json`. No extension is fetched at launch. Linux needs libXcursor alongside the usual X11/XWayland libraries. To run the synthetic X11 integration check, install `scripts/requirements-native-tests.txt` for the synthetic X11 sender and the maintainer-only `xclip` utility for clipboard/picker tests, then run `xvfb-run -a python3 scripts/check-native-drop.py`. It uses disposable fixtures and chooses Keep original. It exercises password-first drops, file/folder-first native pickers, typed/pasted confirmations, mismatches, Clear cancellation, and generated-password confirmation without clicking Process selected. Windows/macOS native drop behavior still needs physical-system testing.

Go toolchain updates do not change already compiled apps. The app has no expiration or required online service. Future OS/library/security-policy changes can require a rebuild; indefinite compatibility cannot be guaranteed. Keep a supported app, your password and a backup of important archives. Format compatibility fixtures and source are preserved for future recovery.
