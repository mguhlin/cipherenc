// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var noCancel = func() error { return nil }
var noProgress = func(float64) {}

func fixtureTree(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Lesson plans")
	if err := os.MkdirAll(filepath.Join(root, "Nested", "Empty"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"Notes.txt": []byte("Exact trailing lines\n\n"), "Nested/zero.bin": {}, "Nested/data.bin": {0, 1, 255, 13, 10}} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
func TestAdjacentArchiveRestoreAndShred(t *testing.T) {
	source := fixtureTree(t)
	p, err := encryptAdjacent(source, testPassword, noCancel, noProgress)
	if err != nil {
		t.Fatal(err)
	}
	defer p.close()
	if p.destination != source+".cen" {
		t.Fatal(p.destination)
	}
	if _, err = os.Stat(source); err != nil {
		t.Fatal("original removed without consent", err)
	}
	restored, err := decryptAdjacent(p.destination, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Notes.txt", "Nested/zero.bin", "Nested/data.bin"} {
		want, _ := os.ReadFile(filepath.Join(source, filepath.FromSlash(name)))
		got, err := os.ReadFile(filepath.Join(restored, "Lesson plans", filepath.FromSlash(name)))
		if err != nil || !bytes.Equal(want, got) {
			t.Fatalf("restore %s: %v", name, err)
		}
	}
	if st, err := os.Stat(filepath.Join(restored, "Lesson plans", "Nested", "Empty")); err != nil || !st.IsDir() {
		t.Fatal("empty directory lost", err)
	}
	// Synthetic originals only. The accessible bytes are overwritten through open handles.
	held, err := os.Open(filepath.Join(source, "Notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if runtime.GOOS == "windows" {
		held.Close()
	}
	if err = p.shred(testPassword); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(source); !os.IsNotExist(err) {
		t.Fatal("original tree remains", err)
	}
	if _, err = os.Stat(p.destination); err != nil {
		t.Fatal("archive removed", err)
	}
	if data, err := readLimited(held, 100); runtime.GOOS != "windows" && err == nil && bytes.Equal(data, []byte("Exact trailing lines\n\n")) {
		t.Fatal("original bytes were not overwritten")
	}
}
func TestAdjacentCollisionAndCancel(t *testing.T) {
	source := filepath.Join(t.TempDir(), "draft.txt")
	os.WriteFile(source, []byte("keep this"), 0600)
	os.WriteFile(source+".cen", []byte("previous archive"), 0600)
	p, err := encryptAdjacent(source, testPassword, noCancel, noProgress)
	if err != nil {
		t.Fatal(err)
	}
	defer p.close()
	if p.destination != source+" (2).cen" {
		t.Fatal(p.destination)
	}
	data, _ := os.ReadFile(source + ".cen")
	if string(data) != "previous archive" {
		t.Fatal("existing archive replaced")
	}
	other := filepath.Join(t.TempDir(), "cancel.txt")
	os.WriteFile(other, []byte("keep"), 0600)
	_, err = encryptAdjacent(other, testPassword, func() error { return errors.New("cancelled") }, noProgress)
	if err == nil {
		t.Fatal("cancellation ignored")
	}
	if _, err = os.Stat(other + ".cen"); !os.IsNotExist(err) {
		t.Fatal("cancelled archive saved")
	}
}
func TestShredRefusesUnsafeChanges(t *testing.T) {
	for _, scenario := range []string{"changed", "new-entry", "tampered-archive", "hardlink", "symlink"} {
		t.Run(scenario, func(t *testing.T) {
			source := fixtureTree(t)
			original := filepath.Join(source, "Notes.txt")
			p, err := encryptAdjacent(source, testPassword, noCancel, noProgress)
			if err != nil {
				t.Fatal(err)
			}
			defer p.close()
			switch scenario {
			case "changed":
				os.WriteFile(original, []byte("new version"), 0600)
			case "new-entry":
				os.WriteFile(filepath.Join(source, "new.txt"), []byte("new"), 0600)
			case "tampered-archive":
				os.WriteFile(p.destination, []byte("corrupt"), 0600)
			case "hardlink":
				if err = os.Link(original, filepath.Join(filepath.Dir(source), "alias.txt")); err != nil {
					t.Skip(err)
				}
			case "symlink":
				if err = os.Symlink(original, filepath.Join(source, "link")); err != nil {
					t.Skip(err)
				}
			}
			before, _ := os.ReadFile(original)
			if err = p.shred(testPassword); err == nil {
				t.Fatal("unsafe shredding permitted")
			}
			after, err := os.ReadFile(original)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("original changed despite refused shredding", err)
			}
		})
	}
}
func TestArchiveInputRejectsLinksAndCaseCollision(t *testing.T) {
	source := fixtureTree(t)
	if err := os.Symlink(filepath.Join(source, "Notes.txt"), filepath.Join(source, "link")); err != nil {
		t.Skip(err)
	}
	if p, data, err := archiveSource(source, noCancel, noProgress); err == nil {
		p.close()
		clear(data)
		t.Fatal("symlink accepted")
	}
	os.Remove(filepath.Join(source, "link"))
	os.WriteFile(filepath.Join(source, "notes.txt"), []byte("other"), 0600)
	// Case-insensitive filesystems cannot construct this fixture.
	entries, _ := os.ReadDir(source)
	if len(entries) == 3 {
		if _, _, err := archiveSource(source, noCancel, noProgress); err == nil {
			t.Fatal("nonportable case collision accepted")
		}
	}
}
func TestZIPRestoreRejectsTraversalDuplicatesAndBomb(t *testing.T) {
	for _, names := range [][]string{{"../escape"}, {"/absolute"}, {"C:/drive"}, {"CON.txt"}, {"a.txt", "A.txt"}, {"link"}} {
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		for _, name := range names {
			h := &zip.FileHeader{Name: name, Method: zip.Deflate}
			h.SetMode(0600)
			if name == "link" {
				h.SetMode(os.ModeSymlink | 0777)
			}
			w, _ := z.CreateHeader(h)
			w.Write([]byte("fixture"))
		}
		z.Close()
		parent := t.TempDir()
		if _, err := restoreZIP(parent, b.Bytes()); err == nil {
			t.Fatalf("unsafe ZIP accepted: %v", names)
		}
		entries, _ := os.ReadDir(parent)
		if len(entries) != 0 {
			t.Fatal("partial restore remained")
		}
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, _ := z.Create("huge")
	w.Write([]byte(strings.Repeat("A", 100)))
	z.Close()
	// Forge the central directory's declared uncompressed size beyond the budget.
	data := b.Bytes()
	index := bytes.Index(data, []byte{0x50, 0x4b, 0x01, 0x02})
	copy(data[index+24:index+28], []byte{0xff, 0xff, 0xff, 0x7f})
	if _, err := restoreZIP(t.TempDir(), data); err == nil {
		t.Fatal("oversized ZIP accepted")
	}
}

func TestAutomaticIntentAndPortableText(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "ordinary.txt")
	os.WriteFile(plain, []byte("not encrypted"), 0600)
	if yes, err := isEncryptedInput(plain); err != nil || yes {
		t.Fatal(yes, err)
	}
	note := "Native and browser text.\n\n"
	packed, err := sealText(note, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	files, err := openPackage(packed, testPassword)
	if err != nil || len(files) != 1 || string(files[0].Data) != note {
		t.Fatal("text envelope round trip", err)
	}
	renamed := filepath.Join(dir, "encrypted-but-renamed.bin")
	os.WriteFile(renamed, packed, 0600)
	if yes, err := isEncryptedInput(renamed); err != nil || !yes {
		t.Fatal("content detection", yes, err)
	}
	invalid := filepath.Join(dir, "misleading.cen")
	os.WriteFile(invalid, []byte("not encrypted"), 0600)
	if _, err := isEncryptedInput(invalid); err == nil {
		t.Fatal("invalid .cen silently re-encrypted")
	}
	packed[len(packed)-3] ^= 1
	if _, err := openPackage(packed, testPassword); err == nil {
		t.Fatal("tampered text accepted")
	}
}

func TestBrowserCompressedArchiveFixture(t *testing.T) {
	data, err := os.ReadFile("reference/compatibility/browser-zip-fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct{ Password, Data string }
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	packed, err := base64.StdEncoding.DecodeString(fixture.Data)
	if err != nil {
		t.Fatal(err)
	}
	files, err := openPackage(packed, fixture.Password)
	if err != nil || len(files) != 1 || files[0].Type != zipMediaType {
		t.Fatal("browser archive metadata", err)
	}
	folder, err := restoreZIP(t.TempDir(), files[0].Data)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(filepath.Join(folder, "Lessons", "one.txt"))
	if err != nil || string(restored) != "Browser ZIP bytes.\n\n" {
		t.Fatal("browser archive bytes", err)
	}
	empty, err := os.ReadFile(filepath.Join(folder, "Lessons", "Nested", "empty.bin"))
	if err != nil || len(empty) != 0 {
		t.Fatal("browser archive empty file", err)
	}
}
