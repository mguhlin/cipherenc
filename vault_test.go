package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const testPassword = "Synthetic independent passphrase!"

func TestNativeRoundTrips(t *testing.T) {
	for _, files := range [][]vaultFile{
		{{Path: "hello.txt", Size: 5, Data: []byte("hello")}},
		{{Path: "folder/note.txt", Size: 6, Data: []byte("sample")}, {Path: "folder/deep/empty.bin", Data: []byte{}}},
		{{Path: "many.bin", Size: chunkSize + 3, Data: bytes.Repeat([]byte{7}, chunkSize+3)}},
	} {
		encrypted, e := sealAES(files, testPassword)
		if e != nil {
			t.Fatal(e)
		}
		opened, e := openPackage(encrypted, testPassword)
		if e != nil {
			t.Fatal(e)
		}
		if len(opened) != len(files) {
			t.Fatal("file count changed")
		}
		for i := range opened {
			if opened[i].Path != files[i].Path || !bytes.Equal(opened[i].Data, files[i].Data) {
				t.Fatal("contents changed")
			}
		}
		again, _ := sealAES(files, testPassword)
		if bytes.Equal(again, encrypted) {
			t.Fatal("random salt/nonce reused")
		}
		if _, e = openPackage(encrypted, "wrong password"); e == nil {
			t.Fatal("wrong password accepted")
		}
		for _, pos := range []int{0, 13, len(encrypted) - 1} {
			changed := bytes.Clone(encrypted)
			changed[pos] ^= 1
			if _, e = openPackage(changed, testPassword); e == nil {
				t.Fatal("mutation accepted")
			}
		}
		for _, changed := range [][]byte{encrypted[:len(encrypted)-1], append(bytes.Clone(encrypted), 0)} {
			if _, e = openPackage(changed, testPassword); e == nil {
				t.Fatal("invalid structure accepted")
			}
		}
	}
}
func TestPathsAndWrites(t *testing.T) {
	for _, name := range []string{"../bad", "/bad", "a/../b", "a\\b", "a\x00b", "a:b", "a//b"} {
		if safeRelative(name) {
			t.Fatal("unsafe path", name)
		}
	}
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.txt")
	if e := saveNew(existing, []byte("first")); e != nil {
		t.Fatal(e)
	}
	if e := saveNew(existing, []byte("second")); e == nil {
		t.Fatal("overwrote existing file")
	}
	content, _ := os.ReadFile(existing)
	if string(content) != "first" {
		t.Fatal("existing content changed")
	}
	if _, e := restoreNewFolder(dir, []vaultFile{{Path: "../escape", Data: []byte("unsafe")}}); e == nil {
		t.Fatal("traversal accepted")
	}
	parent, e := restoreNewFolder(dir, []vaultFile{{Path: "a/b.txt", Data: []byte("safe")}})
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(parent, "a/b.txt"))
	if e != nil || string(b) != "safe" {
		t.Fatal("restore failed")
	}
	if e = os.Symlink(existing, filepath.Join(dir, "link")); e == nil {
		if _, e = collectFiles(filepath.Join(dir, "link")); e == nil {
			t.Fatal("symlink selected")
		}
	}
}
func TestLimitsAndKDF(t *testing.T) {
	if _, e := sealAES([]vaultFile{{Path: "ok", Size: 1, Data: []byte("a")}}, "short"); e == nil {
		t.Fatal("short password accepted")
	}
	for _, h := range []vaultHeader{{KDF: "Argon2id-v1.3", Salt: make([]byte, 16), Memory: 65537, Passes: 3, Parallelism: 1, KeyLength: 32}, {KDF: "PBKDF2-SHA-256", Salt: make([]byte, 16), Iter: 99999}} {
		if _, e := derived(h, testPassword); e == nil {
			t.Fatal("unsafe KDF accepted")
		}
	}
	for _, data := range [][]byte{nil, []byte("CSPCS4!!"), []byte("PRISMVAULT3.invalid")} {
		if _, e := openPackage(data, testPassword); e == nil {
			t.Fatal("malformed package accepted")
		}
	}
}
func TestIndependentLibsodiumFixture(t *testing.T) {
	var fixture struct{ Password, Package, Data string }
	b, e := os.ReadFile("reference/compatibility/independent-xchacha.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &fixture); e != nil {
		t.Fatal(e)
	}
	data, e := base64.StdEncoding.DecodeString(fixture.Package)
	if e != nil {
		t.Fatal(e)
	}
	opened, e := openPackage(data, fixture.Password)
	if e != nil || string(opened[0].Data) != fixture.Data {
		t.Fatal("independent fixture failed", e)
	}
}
func TestNativeSelfTest(t *testing.T) {
	if e := selfTest(t.TempDir()); e != nil {
		t.Fatal(e)
	}
}
func TestCompatibilityFixtures(t *testing.T) {
	b, e := os.ReadFile("reference/compatibility/browser-fixtures.json")
	if os.IsNotExist(e) {
		t.Skip("run scripts/browser-interop.cjs to generate independent browser fixtures")
	}
	if e != nil {
		t.Fatal(e)
	}
	var fixtures []struct{ Name, Password, Data, Expected string }
	if e = json.Unmarshal(b, &fixtures); e != nil {
		t.Fatal(e)
	}
	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			data, e := base64.StdEncoding.DecodeString(f.Data)
			if e != nil {
				t.Fatal(e)
			}
			files, e := openPackage(data, f.Password)
			if e != nil {
				t.Fatal(e)
			}
			if len(files) != 1 || string(files[0].Data) != f.Expected {
				t.Fatal("CipherEnc compatibility failed")
			}
		})
	}
}
