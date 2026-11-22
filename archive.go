// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const zipMediaType = "application/vnd.cipherenc.zip"

type sourceEntry struct {
	name string
	info fs.FileInfo
	hash [32]byte
}
type archivePlan struct {
	root                *os.Root
	source, destination string
	entries             []sourceEntry
	zipHash, cipherHash [32]byte
}

func (p *archivePlan) close() { _ = p.root.Close() }
func portablePath(name string) bool {
	if !safeRelative(name) || strings.ContainsAny(name, "<>\"|?*") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.TrimRight(part, " .") != part {
			return false
		}
		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9' {
			return false
		}
	}
	return true
}
func sourceInventory(root *os.Root, name string) ([]sourceEntry, error) {
	var entries []sourceEntry
	seen := map[string]bool{}
	var total int64
	err := fs.WalkDir(root.FS(), name, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !portablePath(path) {
			return fmt.Errorf("Unsupported filename: %s", path)
		}
		key := strings.ToLower(path)
		if seen[key] {
			return errors.New("Two filenames differ only by letter case. Rename one before encrypting for portable restoration.")
		}
		seen[key] = true
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("Only regular files and folders are supported: %s", path)
		}
		if !info.IsDir() {
			total += info.Size()
		}
		if total > nativeLimit || len(entries) >= maxFiles {
			return errors.New("Choose at most 10,000 entries totaling 256 MiB per archive.")
		}
		entries = append(entries, sourceEntry{name: path, info: info})
		return nil
	})
	return entries, err
}
func readSource(root *os.Root, entry sourceEntry) ([]byte, error) {
	f, err := root.Open(filepath.FromSlash(entry.name))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !sameSource(entry.info, info) {
		return nil, errors.New("An original changed. Close its editor and try again.")
	}
	data, err := readLimited(f, int(entry.info.Size())+1)
	if err == nil && int64(len(data)) != entry.info.Size() {
		err = errors.New("An original changed while reading.")
	}
	return data, err
}
func sameSource(a, b fs.FileInfo) bool {
	return os.SameFile(a, b) && a.Mode() == b.Mode() && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

// archiveSource keeps plaintext ZIP bytes in memory only, including empty directories.
func archiveSource(input string, check func() error, progress func(float64)) (*archivePlan, []byte, error) {
	abs, err := filepath.Abs(filepath.Clean(input))
	if err != nil {
		return nil, nil, err
	}
	base := filepath.Base(abs)
	if base == "." || base == string(filepath.Separator) {
		return nil, nil, errors.New("Choose a file or a folder inside a drive, not a whole drive.")
	}
	root, err := os.OpenRoot(filepath.Dir(abs))
	if err != nil {
		return nil, nil, err
	}
	p := &archivePlan{root: root, source: abs}
	ok := false
	defer func() {
		if !ok {
			p.close()
		}
	}()
	p.entries, err = sourceInventory(root, filepath.ToSlash(base))
	if err != nil {
		return nil, nil, err
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for i := range p.entries {
		if err = check(); err != nil {
			return nil, nil, err
		}
		e := &p.entries[i]
		name := e.name
		mode := fs.FileMode(0600)
		if e.info.IsDir() {
			name += "/"
			mode = os.ModeDir | 0700
		}
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(mode)
		// Do not record device paths, ownership, or source modification timestamps.
		w, err := z.CreateHeader(h)
		if err != nil {
			return nil, nil, err
		}
		if !e.info.IsDir() {
			data, err := readSource(root, *e)
			if err != nil {
				return nil, nil, err
			}
			e.hash = sha256.Sum256(data)
			_, err = w.Write(data)
			clear(data)
			if err != nil {
				return nil, nil, err
			}
		}
		if buf.Len() > nativeLimit {
			return nil, nil, errors.New("The ZIP exceeds the 256 MiB archive limit.")
		}
		progress(float64(i+1) / float64(len(p.entries)) * 40)
	}
	if err = z.Close(); err != nil {
		return nil, nil, err
	}
	if buf.Len() > nativeLimit {
		return nil, nil, errors.New("The ZIP exceeds the 256 MiB archive limit.")
	}
	p.zipHash = sha256.Sum256(buf.Bytes())
	ok = true
	return p, buf.Bytes(), nil
}
func archiveName(base, suffix string) string {
	// Leave room for numbering and the extension, including on Windows.
	for len(base) > 210 {
		base = string([]rune(base)[:len([]rune(base))-1])
	}
	return base + suffix
}
func (p *archivePlan) save(data []byte) error {
	base := filepath.Base(p.source)
	for n := 1; n <= 9999; n++ {
		suffix := ".cen"
		if n > 1 {
			suffix = fmt.Sprintf(" (%d).cen", n)
		}
		name := archiveName(base, suffix)
		f, err := p.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = p.root.Remove(name)
			return err
		}
		p.destination = filepath.Join(filepath.Dir(p.source), name)
		p.cipherHash = sha256.Sum256(data)
		return nil
	}
	return errors.New("Too many existing archives beside this original.")
}
func (p *archivePlan) verify(password string) error {
	f, err := p.root.Open(filepath.Base(p.destination))
	if err != nil {
		return err
	}
	data, err := readLimited(f, nativeLimit+(16<<20))
	f.Close()
	if err != nil {
		return err
	}
	defer clear(data)
	if sha256.Sum256(data) != p.cipherHash {
		return errors.New("The saved archive changed; originals must be kept.")
	}
	files, err := openPackage(data, password)
	if err != nil {
		return err
	}
	defer func() {
		for i := range files {
			clear(files[i].Data)
		}
	}()
	if len(files) != 1 || files[0].Type != zipMediaType || sha256.Sum256(files[0].Data) != p.zipHash {
		return errors.New("Saved archive verification failed; originals must be kept.")
	}
	return nil
}
func encryptAdjacent(input, password string, check func() error, progress func(float64)) (*archivePlan, error) {
	p, data, err := archiveSource(input, check, progress)
	if err != nil {
		return nil, err
	}
	defer clear(data)
	ok := false
	defer func() {
		if !ok {
			p.close()
		}
	}()
	progress(45)
	encrypted, err := sealAES([]vaultFile{{Path: archiveName(filepath.Base(input), ".zip"), Type: zipMediaType, Size: len(data), Data: data}}, password)
	if err != nil {
		return nil, err
	}
	defer clear(encrypted)
	if err = check(); err != nil {
		return nil, err
	}
	progress(70)
	if err = p.save(encrypted); err != nil {
		return nil, err
	}
	progress(80)
	if err = p.verify(password); err != nil {
		return nil, fmt.Errorf("Archive saved at %s, but verification failed: %w", p.destination, err)
	}
	progress(100)
	ok = true
	return p, nil
}

// shred is called only after explicit GUI confirmation and fresh archive verification.
// It overwrites accessible file bytes once, syncs, and unlinks. SSD/COW/backups can
// retain other copies. There is deliberately no recursive RemoveAll on originals.
func (p *archivePlan) shred(password string) error {
	parent, err := os.Stat(filepath.Dir(p.source))
	if err != nil {
		return err
	}
	opened, err := p.root.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(parent, opened) {
		return errors.New("The original's parent folder moved; shredding stopped.")
	}
	if err := p.verify(password); err != nil {
		return err
	}
	current, err := sourceInventory(p.root, filepath.ToSlash(filepath.Base(p.source)))
	if err != nil {
		return err
	}
	if len(current) != len(p.entries) {
		return errors.New("Original folder contents changed; nothing was shredded.")
	}
	for i, e := range current {
		old := p.entries[i]
		if e.name != old.name || !sameSource(e.info, old.info) {
			return errors.New("An original changed; nothing was shredded.")
		}
		if !e.info.IsDir() {
			f, err := p.root.Open(filepath.FromSlash(e.name))
			if err != nil {
				return err
			}
			info, err := f.Stat()
			if err == nil {
				err = singleLink(f, info)
			}
			if err != nil {
				f.Close()
				return err
			}
			data, err := readLimited(f, int(e.info.Size())+1)
			f.Close()
			hash := sha256.Sum256(data)
			clear(data)
			if err != nil {
				return err
			}
			if hash != old.hash {
				return errors.New("An original changed; nothing was shredded.")
			}
		}
	}
	block := make([]byte, 64<<10)
	defer clear(block)
	for i := len(p.entries) - 1; i >= 0; i-- {
		e := p.entries[i]
		path := filepath.FromSlash(e.name)
		info, err := p.root.Lstat(path)
		if err != nil {
			return err
		}
		// Directory timestamps change as their children are removed; identity must not.
		if !os.SameFile(info, e.info) || info.Mode() != e.info.Mode() {
			return errors.New("An original was replaced; shredding stopped.")
		}
		if e.info.IsDir() {
			if err = p.root.Remove(path); err != nil {
				return err
			}
			continue
		}
		f, err := p.root.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			return err
		}
		err = func() error {
			info, err := f.Stat()
			if err != nil {
				return err
			}
			if !sameSource(e.info, info) {
				return errors.New("An original changed; shredding stopped.")
			}
			if err = singleLink(f, info); err != nil {
				return err
			}
			data, err := readLimited(f, int(info.Size())+1)
			hash := sha256.Sum256(data)
			clear(data)
			if err != nil {
				return err
			}
			if hash != e.hash {
				return errors.New("An original changed; shredding stopped.")
			}
			if _, err = f.Seek(0, io.SeekStart); err != nil {
				return err
			}
			for left := info.Size(); left > 0; {
				n := int64(len(block))
				if left < n {
					n = left
				}
				if _, err = rand.Read(block[:n]); err != nil {
					return err
				}
				if _, err = f.Write(block[:n]); err != nil {
					return err
				}
				left -= n
			}
			if err = f.Sync(); err != nil {
				return err
			}
			now, err := p.root.Lstat(path)
			if err != nil {
				return err
			}
			if !os.SameFile(now, info) {
				return errors.New("An original path changed; shredding stopped.")
			}
			return nil
		}()
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if err = p.root.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

func restoreZIP(parent string, data []byte) (string, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	if len(z.File) > maxFiles {
		return "", errors.New("Too many ZIP entries.")
	}
	seen := map[string]bool{}
	var total uint64
	for _, f := range z.File {
		name := strings.TrimSuffix(f.Name, "/")
		key := strings.ToLower(name)
		if !portablePath(name) || seen[key] || f.Mode()&os.ModeSymlink != 0 || !f.FileInfo().IsDir() && !f.Mode().IsRegular() {
			return "", errors.New("Unsafe or duplicate ZIP entry.")
		}
		seen[key] = true
		if f.UncompressedSize64 > nativeLimit-total {
			return "", errors.New("Restored ZIP exceeds 256 MiB.")
		}
		total += f.UncompressedSize64
		if total > nativeLimit {
			return "", errors.New("Restored ZIP exceeds 256 MiB.")
		}
	}
	folder, err := os.MkdirTemp(parent, "CipherEnc-restored-")
	if err != nil {
		return "", err
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(folder)
		}
	}()
	root, err := os.OpenRoot(folder)
	if err != nil {
		return "", err
	}
	defer root.Close()
	for _, f := range z.File {
		name := filepath.FromSlash(strings.TrimSuffix(f.Name, "/"))
		if f.FileInfo().IsDir() {
			err = root.MkdirAll(name, 0700)
		} else {
			err = root.MkdirAll(filepath.Dir(name), 0700)
			if err != nil {
				return "", err
			}
			input, e := f.Open()
			if e != nil {
				return "", e
			}
			content, e := readLimited(input, int(f.UncompressedSize64)+1)
			input.Close()
			if e != nil {
				return "", e
			}
			if uint64(len(content)) != f.UncompressedSize64 {
				clear(content)
				return "", errors.New("Invalid ZIP entry size.")
			}
			output, e := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if e != nil {
				clear(content)
				return "", e
			}
			_, err = output.Write(content)
			clear(content)
			closeErr := output.Close()
			if err == nil {
				err = closeErr
			}
		}
		if err != nil {
			return "", err
		}
	}
	success = true
	return folder, nil
}
func decryptAdjacent(input, password string) (string, error) {
	f, err := os.Open(input)
	if err != nil {
		return "", err
	}
	data, err := readLimited(f, nativeLimit+(16<<20))
	f.Close()
	if err != nil {
		return "", err
	}
	files, err := openPackage(data, password)
	clear(data)
	if err != nil {
		return "", err
	}
	defer func() {
		for i := range files {
			clear(files[i].Data)
		}
	}()
	if len(files) == 1 && files[0].Type == zipMediaType {
		return restoreZIP(filepath.Dir(input), files[0].Data)
	}
	return restoreNewFolder(filepath.Dir(input), files)
}

// Decide from container bytes first; a misleading .cen name is never silently
// re-encrypted. Unknown extensions on a recognized package are safe to restore.
func isEncryptedInput(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() && !info.Mode().IsRegular() {
		return false, errors.New("Choose regular files or folders, not links or device files.")
	}
	if info.IsDir() {
		return false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	var prefix [32]byte
	n, err := f.Read(prefix[:])
	if err != nil && err != io.EOF {
		return false, err
	}
	head := string(prefix[:n])
	for _, magic := range []string{"CSPCS4!!", "CSPCS3!!", "CSPCS2!!", "NIVORA1!", "CENCX01!", "PRISMVAULT1.", "PRISMVAULT2.", "PRISMVAULT3."} {
		if strings.HasPrefix(head, magic) {
			return true, nil
		}
	}
	if strings.EqualFold(filepath.Ext(path), ".cen") {
		return false, errors.New("This .cen file is not a supported desktop package. OpenPGP packages need the browser tool.")
	}
	return false, nil
}
func onlyEncrypted(paths []string) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		yes, err := isEncryptedInput(path)
		if err != nil || !yes {
			return false
		}
	}
	return true
}
