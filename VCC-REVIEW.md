# CipherEnc VCC release review

All four primary VCC skills were read before implementation and compared byte-for-byte with their published copies: [Web Design](https://mguhlin.github.io/vcc/agent-skills/vcc-web-design/SKILL.md), [SecureVCC](https://mguhlin.github.io/vcc/agent-skills/securevcc/SKILL.md), [Education Privacy](https://mguhlin.github.io/vcc/agent-skills/edu-compliance/SKILL.md), and [Vibe-Code Rescue](https://mguhlin.github.io/vcc/agent-skills/vibe-code-rescue/SKILL.md). This is an engineering review using synthetic fixtures, not an independent security audit or certification.

## Product and design

The main tasks are adding local files/folders or text, protecting them with a password, automatically saving a verified .cen package, and restoring it without overwriting an original. General desktop users and educators are the inferred audience; ages and institutional authorization are unknown. The desktop app uses native controls in a blue/silver/navy palette. The download site presents platform choices, real Linux native-window captures, a generated banner, a three-panel comic with text equivalents, a browser tool, and honest format/platform differences.

The unified workspace detects file/text intent from supported container contents. Its main interface has no mode/action/integrity tabs. ZIP compression and authenticated verification are integrated. Desktop file output is created beside the source; destruction is a separate post-verification prompt with Keep original as the default. The site is a task-first download and learning page. Native controls support keyboard navigation; website engineering target is WCAG 2.2 AA. Full screen-reader conformance is not established.

## Scope and evidence

| Check | Actual evidence and result |
| --- | --- |
| Desktop architecture | Native Go/Tk window with embedded runtime, no browser dependency or listener; source reviewed and Linux executable/UI exercised |
| CipherEnc compatibility | Browser-generated CS2, CS3, CS4 and text-v1/v2/v3 fixtures opened by Go; independent browser engine restores native Go CS4 nested-folder output |
| Crypto and rejection | Go race/unit tests cover round trips, random salt/nonce changes, wrong passwords, altered headers/ciphertext/tags, malformed/truncated/appended data, limits and unsafe paths |
| XChaCha independent fixture | Go opens a fixture generated through system libargon2 and libsodium; the app uses the pinned Go crypto implementation |
| Archive and shredding | ZIP exists only in memory before encryption; saved ciphertext is reopened/authenticated/compared before a prompt. Synthetic tests cover empty directories, collisions, byte preservation, hard links, source changes, archive tampering, ZIP traversal and expansion bounds. Best-effort shredding is explicit, defaults to keeping originals, and makes no SSD/backup erasure claim; browser/Android do not shred |
| Restoration authority | Full package authentication precedes Save; existing files are never replaced; restored folders are new directories confined by os.Root; traversal and overwrite cases tested |
| UI lifecycle | Actual native window opens and completes a synthetic text encryption/restoration; credentials clear, controls lock during work, and result references can be cleared. A fresh source review found native note newline trimming; it was removed and actual Tk UI input with two trailing newlines is verified byte-for-byte |
| Offline operation | Native binaries cross-built in a network namespace with no internet route; native runtime and browser crypto checks use isolated loopback or Xvfb; application has no remote service path |
| Dependency scanning | govulncheck flagged a transitive image decoder; x/image updated from v0.36.0 to v0.48.0 and scan repeated with no reachable vulnerabilities reported. Other imported/module advisory leads remain distinguished from reachable behavior |
| Browser engine | CipherEnc functionality retained with .cen filenames, strict CSP, local vendors, safe text rendering, no operator uploads or app persistence; restoration unzips authenticated packages, saves a single original directly, and offers individual files or an explicit ZIP alternative where directory access is unavailable |
| Browser accessibility | Chromium/Firefox/WebKit task and viewport tests plus axe-core sampling; inherited low-contrast controls/labels corrected. Automation does not establish full conformance |
| Android | Installable evaluation APK built and signature/manifest inspected; bundled assets, no INTERNET permission, trusted-origin native bridge, bounded SAF operations. Device/emulator runtime is NOT RUN |
| Source/setup | Pinned dependencies, offline release source archive, explicit build/test scripts, component responsibilities and recovery guidance in README; private signing material excluded |
| Hosted records, accounts, payments, AI | N/A: none are present |

## Privacy and school review

The local utility has user-controlled exports and potentially sensitive user-selected data: privacy mode B. Native inputs and keys remain in process memory; browser inputs stay in browser memory. Website hosting receives ordinary asset requests. The Android document provider may be another app, including a cloud provider chosen by the user. These are different data paths and are disclosed in README/interface documentation.

No real student information is used in fixtures, screenshots or public materials. Clear workspace and idle clearing release active references, not downloads, clipboard contents, OS copies or guaranteed physical memory erasure. Native GUI runtime libraries may persist in a private user cache; document content and passwords are not saved there by the app.

Institutional authorization, geography, user ages, applicable contracts and legal review are UNVERIFIED. The [U.S. Department of Education](https://studentprivacy.ed.gov/faq/i-want-use-online-tool-or-application-part-my-course-however-i-am-worried-it-violation-ferpa) directs educators to consult their administration and IT representatives about classroom tools. No school approval, DPA, legal compliance, accessibility certification or formal cryptographic certification is invented.

## Release gate and limitations

Desktop/source/browser checks support a technical pilot with synthetic data. The Android artifact is explicitly an evaluation build, development-signed and untested on physical hardware or an emulator. Native Windows/macOS physical launch, ARM64 hardware, iOS/Android physical browsers, VoiceOver/TalkBack, full translation review, and very large data/memory stress cases are not established by Linux tests. GitHub runner checks and live publication are observed separately from local results.

The browser tool supports AES/OpenPGP; the native desktop app supports AES/XChaCha20. XChaCha20 packages are a separate CipherEnc format and do not claim Paranoia C4 equivalence. A broad new collection of legacy or custom ciphers was not introduced as an unsupported security claim. No critical exposure was confirmed in the reviewed scope; this is not a warranty or a claim that every issue was found.

## Unified workspace evidence

The generated-password Copy handler previously accessed `event.currentTarget` after awaiting the clipboard request, falsely reporting a blocked clipboard even after a successful write. The handler retains its button reference; synthetic async-success, denied, unavailable, manual-selection and direct-fill cases cover all three browser engines and Android/iPhone viewport profiles. The browser save workflow preserves the picker gesture before encryption and retains results on write failure; cancellation, verification-before-write and fallback downloads are tested. Ordinary file inputs and encrypted packages use the same workspace, as does text. All cryptographic fixtures and shred tests use disposable synthetic content.

The native app embeds TkDND for six desktop targets. Library origins and hashes are pinned; private temporary extraction contains application code only. Linux X11 native protocol delivery from a separate synthetic source is tested separately from browser synthetic event/picker tests. Cross-build success does not establish Windows/macOS physical drag behavior. The browser's automatic download request cannot override a user's download settings or every mobile browser policy; a save-again control remains available.

ZIP restoration validates its central directory before decompression, rejects unsafe/duplicate paths and links, bounds the actual decompression stream, and checks every file size and CRC. No plaintext is exported until the entire archive has passed these checks. New-directory restoration preserves empty folders where the browser exposes directory access. Readable numbered instructions and a shared C-shield icon accompany the larger web type scale; encrypted and decrypted text are available in a copyable on-screen result box.

The dedicated OpenPGP page is derived from the shared workspace and keeps imported keys in tab memory. Synthetic interface tests cover armored/binary key files, local fingerprint display, protected private keys, retry after an incorrect passphrase, exact text and original-filename restoration, refusal of private keys as encryption recipients, misplaced keys/password envelopes, oversized keys and key URLs without server fetching. Chromium, Firefox, WebKit and mobile viewport profiles pass these flows, sampled axe checks and narrow-screen layout checks. The Android adapter opens key controls inline without page navigation. These tests do not establish physical mobile behavior or add OpenPGP support to the native Go edition.

An isolated synthetic GnuPG keyring independently decrypted an OpenPGP message produced through CipherEnc’s interface and restored its exact text. No personal keyring or real user data was used. A delayed key-file read is invalidated by Clear so it cannot repopulate credentials afterward.
