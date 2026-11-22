# CipherEnc third-party notices

- Imported encryption engine and shared language controls: DrawSplat Developer(s), from https://github.com/mguhlin/drawsplat, AGPL-3.0-or-later. The imported revision is recorded in `upstream.json`. Original license and brand notices are preserved in `UPSTREAM-LICENSE` and `UPSTREAM-NOTICE.md`. CipherEnc changes the launcher, interface presentation, online controls, encrypted filename suffixes, and saved-text import handling; cryptographic primitives remain supplied by their libraries.
- OpenPGP.js 6.3.1: https://github.com/openpgpjs/openpgpjs, LGPL-3.0-or-later. The distributed library, package metadata, LGPL text, and incorporated GPL text are in `docs/encrypt/vendor/openpgp`. The library bundle is embedded without modification. Corresponding upstream source is included in `third-party-source/openpgpjs-v6.3.1.tar.gz` in the source ZIP.
- hash-wasm 4.12.0: https://github.com/Daninet/hash-wasm, MIT. Copyright Dani Biró and contributors. License, package metadata, and unmodified Argon2 bundle are in `docs/encrypt/vendor/hash-wasm`.
- Go runtime and standard library: The Go Authors, BSD-style license, included as `GO-LICENSE`.
- `golang.org/x/crypto` v0.59.0 and `golang.org/x/sys` v0.50.0: The Go Authors, BSD-3-Clause. Required source and license files are under `vendor/golang.org/x/`; exact versions and sums are in `go.mod`, `go.sum`, and `vendor/modules.txt`.
- VCC review instructions by Miguel Guhlin: https://mguhlin.github.io/vcc/. Web Design and Education Privacy prose are CC BY 4.0; SecureVCC includes MIT notices; Vibe-Code Rescue retains its source-specific terms. These instructions guided review; the instructions themselves are not redistributed in the application.

Playwright and axe-core are development test tools, not runtime dependencies. Their notices remain in their installed packages; the source package records their versions in `package-lock.json`.

Users retain ownership and control of their files, text, keys, and outputs. App licensing does not license user content.

Native GUI: modernc.org/tk9.0 v1.80.1, BSD-3-Clause, with bundled Tcl/Tk runtime notices in vendor/modernc.org/tk9.0/LICENSE-TCLTK. PureGo fork and all Go dependency notices are collected under Support/Licenses/Go-dependencies in the distribution. x/image is pinned at v0.48.0 after the dependency review.

JSZip 3.10.1: MIT, locally bundled for browser folder restoration as a single ZIP. Its license remains beside the browser vendor asset.

Native drag-and-drop: TkDND by Georgios Petasis and contributors, with Mac portions by Kevin Walzer/WordTech Communications LLC. The complete permissive notice is in `internal/tkdnd/LICENSE` and distributed as `Support/Licenses/TkDND-LICENSE`. Bundled Tcl-9-compatible packages come from pinned TkDND builds, including Squiblydoo's portability fork and the upstream Intel Mac release. Platform origins, artifact identifiers and SHA-256 hashes are in `internal/tkdnd/manifest.json`; corresponding source archives are included there. Bundled files are extracted into a private temporary directory at launch and removed where the operating system permits; loaded-library files can remain until OS cleanup. They contain application code, never user documents or passwords.
