// SPDX-License-Identifier: AGPL-3.0-or-later
// CipherSplat CS4/CS3/CS2 and text containers, ported from the upstream app.js.
package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

const nativeLimit = 256 << 20
const chunkSize = 4 << 20
const maxFiles = 10000

type vaultFile struct {
	Path   string `json:"path"`
	Size   int    `json:"size"`
	Type   string `json:"type"`
	Chunks int    `json:"chunks"`
	Data   []byte `json:"-"`
}
type metadata struct {
	Kind    string      `json:"kind"`
	Name    string      `json:"name"`
	Files   []vaultFile `json:"files"`
	Entries []vaultFile `json:"entries,omitempty"`
}
type vaultHeader struct {
	V           int    `json:"v"`
	Alg         string `json:"alg"`
	KDF         string `json:"kdf"`
	Memory      int    `json:"memory,omitempty"`
	Passes      int    `json:"passes,omitempty"`
	Parallelism int    `json:"parallelism,omitempty"`
	KeyLength   int    `json:"keyLength,omitempty"`
	Iter        int    `json:"iter,omitempty"`
	ChunkSize   int    `json:"chunkSize,omitempty"`
	Salt        []byte `json:"salt"`
	Prefix      []byte `json:"prefix,omitempty"`
	IV          []byte `json:"iv,omitempty"`
}

func safeRelative(s string) bool {
	if !utf8.ValidString(s) || s == "" || len(s) > 4096 || strings.ContainsAny(s, "\\\x00\r\n:") {
		return false
	}
	for _, part := range strings.Split(s, "/") {
		if part == "" || part == "." || part == ".." || len(part) > 255 {
			return false
		}
	}
	return true
}
func collectFiles(input string) ([]vaultFile, error) {
	st, err := os.Lstat(input)
	if err != nil {
		return nil, err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("Choose a regular file or folder, not a symbolic link.")
	}
	root := filepath.Dir(input)
	var result []vaultFile
	total := int64(0)
	err = filepath.WalkDir(input, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("Folder contains a symbolic link; remove it from the selection.")
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() {
			return errors.New("Only regular files are supported.")
		}
		total += info.Size()
		if total > nativeLimit || len(result) >= maxFiles {
			return errors.New("Choose at most 10,000 files totaling 256 MiB per package.")
		}
		rel, e := filepath.Rel(root, p)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		if !safeRelative(rel) {
			return errors.New("An unsupported relative filename was found.")
		}
		result = append(result, vaultFile{Path: rel, Size: int(info.Size()), Type: "application/octet-stream"})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, errors.New("The folder contains no files. Empty folders are not recorded.")
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	// Read bounded content only after every entry and total size has been checked.
	confined, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer confined.Close()
	for i := range result {
		file, e := confined.Open(filepath.FromSlash(result[i].Path))
		if e != nil {
			return nil, e
		}
		data, e := io.ReadAll(io.LimitReader(file, int64(result[i].Size)+1))
		file.Close()
		if e != nil {
			return nil, e
		}
		if len(data) != result[i].Size {
			return nil, errors.New("A selected file changed during reading; try again.")
		}
		result[i].Data = data
	}
	return result, nil
}
func derived(h vaultHeader, password string) ([]byte, error) {
	if len(password) == 0 || len(password) > 4096 || len(h.Salt) != 16 {
		return nil, errors.New("Invalid password or salt.")
	}
	if h.KDF == "Argon2id-v1.3" {
		if h.Memory != 65536 || h.Passes != 3 || h.Parallelism != 1 || h.KeyLength != 32 {
			return nil, errors.New("Unsupported or unsafe Argon2id parameters.")
		}
		return argon2.IDKey([]byte(password), h.Salt, 3, 65536, 1, 32), nil
	}
	if h.KDF != "PBKDF2-SHA-256" || h.Iter < 100000 || h.Iter > 2000000 {
		return nil, errors.New("Unsupported or unsafe key parameters.")
	}
	return pbkdf2.Key([]byte(password), h.Salt, h.Iter, 32, sha256.New), nil
}
func gcmFor(h vaultHeader, password string) (cipher.AEAD, error) {
	key, e := derived(h, password)
	if e != nil {
		return nil, e
	}
	defer clear(key)
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(block)
}
func u32(n int) []byte { b := make([]byte, 4); binary.BigEndian.PutUint32(b, uint32(n)); return b }
func recordNonce(prefix []byte, index int) []byte {
	n := make([]byte, 12)
	copy(n, prefix)
	binary.BigEndian.PutUint32(n[8:], uint32(index))
	return n
}
func aadFor(format int, base []byte, index int) []byte {
	if format == 2 {
		return []byte(fmt.Sprintf("NV2:%d", index))
	}
	return append(append([]byte{}, base...), u32(index)...)
}
func sealAES(files []vaultFile, password string) ([]byte, error) {
	if len([]rune(password)) < 12 {
		return nil, errors.New("Use a password of at least 12 characters.")
	}
	if len(files) == 0 || len(files) > maxFiles {
		return nil, errors.New("Invalid file selection.")
	}
	var total int
	seen := map[string]bool{}
	for i := range files {
		f := &files[i]
		if !safeRelative(f.Path) || f.Size != len(f.Data) || seen[strings.ToLower(f.Path)] {
			return nil, errors.New("Unsafe, duplicate, or inconsistent source paths.")
		}
		seen[strings.ToLower(f.Path)] = true
		total += f.Size
		if total > nativeLimit {
			return nil, errors.New("Package exceeds 256 MiB.")
		}
		f.Chunks = (f.Size + chunkSize - 1) / chunkSize
	}
	h := vaultHeader{V: 4, Alg: "AES-256-GCM-CHUNKED", KDF: "Argon2id-v1.3", Memory: 65536, Passes: 3, Parallelism: 1, KeyLength: 32, ChunkSize: chunkSize, Salt: make([]byte, 16), Prefix: make([]byte, 8)}
	if _, e := rand.Read(h.Salt); e != nil {
		return nil, e
	}
	if _, e := rand.Read(h.Prefix); e != nil {
		return nil, e
	}
	header, _ := json.Marshal(h)
	base := append(append([]byte("CSPCS4!!"), u32(len(header))...), header...)
	aead, e := gcmFor(h, password)
	if e != nil {
		return nil, e
	}
	m := metadata{Kind: "file", Name: files[0].Path, Files: files}
	if len(files) > 1 || strings.Contains(files[0].Path, "/") {
		m.Kind = "folder"
		m.Name = strings.Split(files[0].Path, "/")[0]
	}
	meta, _ := json.Marshal(m)
	if len(meta) > 8<<20 {
		return nil, errors.New("Excessive folder metadata.")
	}
	out := bytes.NewBuffer(append([]byte{}, base...))
	counter := 0
	seal := func(data []byte) {
		encrypted := aead.Seal(nil, recordNonce(h.Prefix, counter), data, aadFor(4, base, counter))
		out.Write(u32(len(data)))
		out.Write(u32(len(encrypted)))
		out.Write(encrypted)
		counter++
	}
	seal(meta)
	for _, f := range files {
		for off := 0; off < len(f.Data); off += chunkSize {
			end := min(off+chunkSize, len(f.Data))
			seal(f.Data[off:end])
		}
	}
	return out.Bytes(), nil
}
func openAES(data []byte, password string) ([]vaultFile, error) {
	if len(data) < 29 || len(data) > nativeLimit+16<<20 {
		return nil, errors.New("Invalid or oversized CipherSplat package.")
	}
	magic := string(data[:8])
	format := 0
	switch magic {
	case "CSPCS4!!":
		format = 4
	case "CSPCS3!!":
		format = 3
	case "NIVORA2!":
		format = 2
	default:
		return openText(data, password)
	}
	hl := int(binary.BigEndian.Uint32(data[8:12]))
	if hl < 2 || hl > 4096 || 12+hl+24 > len(data) {
		return nil, errors.New("Invalid package header.")
	}
	header := data[12 : 12+hl]
	var h vaultHeader
	if json.Unmarshal(header, &h) != nil || h.V != format || h.Alg != "AES-256-GCM-CHUNKED" || h.ChunkSize < 65536 || h.ChunkSize > 8<<20 || len(h.Prefix) != 8 {
		return nil, errors.New("Unsupported package header.")
	}
	if format == 4 && h.KDF != "Argon2id-v1.3" || format != 4 && h.KDF != "PBKDF2-SHA-256" {
		return nil, errors.New("Format and key derivation do not match.")
	}
	aead, e := gcmFor(h, password)
	if e != nil {
		return nil, e
	}
	base := data[:12+hl]
	offset := 12 + hl
	counter := 0
	record := func(limit int) ([]byte, error) {
		if offset+8 > len(data) {
			return nil, errors.New("Truncated package.")
		}
		pl := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		cl := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if pl > limit || cl != pl+16 || cl > len(data)-offset {
			return nil, errors.New("Invalid record length.")
		}
		plain, e := aead.Open(nil, recordNonce(h.Prefix, counter), data[offset:offset+cl], aadFor(format, base, counter))
		if e != nil {
			return nil, errUnlock
		}
		offset += cl
		counter++
		return plain, nil
	}
	plain, e := record(8 << 20)
	if e != nil {
		return nil, e
	}
	defer clear(plain)
	var m metadata
	if json.Unmarshal(plain, &m) != nil || m.Kind != "file" && m.Kind != "folder" || len(m.Files) == 0 || len(m.Files) > maxFiles {
		return nil, errors.New("Invalid authenticated metadata.")
	}
	total := 0
	seen := map[string]bool{}
	for _, f := range m.Files {
		if !safeRelative(f.Path) || f.Size < 0 || f.Size > nativeLimit || f.Chunks != (f.Size+h.ChunkSize-1)/h.ChunkSize || seen[strings.ToLower(f.Path)] {
			return nil, errors.New("Unsafe or inconsistent file metadata.")
		}
		seen[strings.ToLower(f.Path)] = true
		total += f.Size
		if total > nativeLimit {
			return nil, errors.New("Restored package exceeds 256 MiB.")
		}
	}
	for i := range m.Files {
		f := &m.Files[i]
		for c := 0; c < f.Chunks; c++ {
			p, e := record(h.ChunkSize)
			if e != nil {
				return nil, e
			}
			f.Data = append(f.Data, p...)
			clear(p)
		}
		if len(f.Data) != f.Size {
			return nil, errors.New("Restored size does not match authenticated metadata.")
		}
	}
	if offset != len(data) {
		return nil, errors.New("Unexpected trailing data.")
	}
	return m.Files, nil
}
func openText(data []byte, password string) ([]vaultFile, error) {
	if len(data) < 29 {
		return nil, errors.New("Not a supported encrypted package.")
	}
	magic := string(data[:8])
	version := map[string]int{"NIVORA1!": 1, "CSPTXT2!": 2, "CSPTXT3!": 3}[magic]
	if version == 0 {
		return nil, errors.New("Not a CipherEnc or CipherSplat package. OpenPGP is not supported in this native edition.")
	}
	hl := int(binary.BigEndian.Uint32(data[8:12]))
	if hl < 2 || hl > 4096 || 12+hl+16 >= len(data) {
		return nil, errors.New("Invalid text header.")
	}
	var h vaultHeader
	if json.Unmarshal(data[12:12+hl], &h) != nil || h.V != version || h.Alg != "AES-256-GCM" || len(h.IV) != 12 {
		return nil, errors.New("Invalid text parameters.")
	}
	if version == 3 && h.KDF != "Argon2id-v1.3" || version != 3 && h.KDF != "PBKDF2-SHA-256" {
		return nil, errors.New("Unsupported text key parameters.")
	}
	aead, e := gcmFor(h, password)
	if e != nil {
		return nil, e
	}
	var aad []byte
	if version > 1 {
		aad = data[:12+hl]
	}
	plain, e := aead.Open(nil, h.IV, data[12+hl:], aad)
	if e != nil {
		return nil, errUnlock
	}
	defer clear(plain)
	if len(plain) < 4 {
		return nil, errors.New("Invalid text contents.")
	}
	ml := int(binary.BigEndian.Uint32(plain[:4]))
	if ml < 2 || ml > 8<<20 || ml > len(plain)-4 {
		return nil, errors.New("Invalid text metadata.")
	}
	var m metadata
	if json.Unmarshal(plain[4:4+ml], &m) != nil {
		return nil, errors.New("Invalid text metadata.")
	}
	content := plain[4+ml:]
	if m.Kind == "text" {
		return []vaultFile{{Path: "private-note.txt", Size: len(content), Type: "text/plain", Data: bytes.Clone(content)}}, nil
	}
	if m.Kind != "file" && m.Kind != "folder" || len(m.Entries) == 0 || len(m.Entries) > maxFiles {
		return nil, errors.New("Invalid legacy file metadata.")
	}
	off := 0
	for i := range m.Entries {
		f := &m.Entries[i]
		if !safeRelative(f.Path) || f.Size < 0 || f.Size > len(content)-off {
			return nil, errors.New("Invalid legacy path or size.")
		}
		f.Data = bytes.Clone(content[off : off+f.Size])
		off += f.Size
	}
	if off != len(content) {
		return nil, errors.New("Invalid legacy payload size.")
	}
	return m.Entries, nil
}
func openPackage(data []byte, password string) ([]vaultFile, error) {
	if len(data) > nativeLimit+(16<<20) {
		return nil, errors.New("Package exceeds the 272 MiB import limit.")
	}
	if bytes.HasPrefix(data, []byte("PRISMVAULT")) {
		s := strings.TrimSpace(string(data))
		parts := strings.SplitN(s, ".", 2)
		if len(parts) != 2 {
			return nil, errors.New("Invalid encrypted text.")
		}
		decoded, e := base64.StdEncoding.DecodeString(parts[1])
		if e != nil {
			return nil, errors.New("Invalid encrypted text encoding.")
		}
		data = decoded
	}
	if bytes.HasPrefix(data, []byte(xMagic)) {
		p, e := openX(data, password)
		if e != nil {
			return nil, e
		}
		return []vaultFile{{Path: p.Name, Size: len(p.Data), Data: p.Data}}, nil
	}
	return openAES(data, password)
}
func saveNew(name string, data []byte) error {
	f, e := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return fmt.Errorf("Choose a new output name; existing files are not replaced: %w", e)
	}
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(name)
		}
	}()
	if _, e = f.Write(data); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	ok = true
	return nil
}
func restoreNewFolder(parent string, files []vaultFile) (string, error) {
	folder, e := os.MkdirTemp(parent, "CipherEnc-restored-")
	if e != nil {
		return "", e
	}
	success := false
	defer func() {
		if !success {
			os.RemoveAll(folder)
		}
	}()
	root, e := os.OpenRoot(folder)
	if e != nil {
		return "", e
	}
	defer root.Close()
	for _, f := range files {
		if !safeRelative(f.Path) {
			return "", errors.New("Unsafe restoration path.")
		}
		name := filepath.FromSlash(f.Path)
		if e = root.MkdirAll(filepath.Dir(name), 0700); e != nil {
			return "", e
		}
		output, e := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return "", e
		}
		_, e = output.Write(f.Data)
		closeErr := output.Close()
		if e != nil {
			return "", e
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	success = true
	return folder, nil
}
