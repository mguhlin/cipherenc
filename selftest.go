package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

func readLimited(r io.Reader, max int) ([]byte, error) {
	data, e := io.ReadAll(io.LimitReader(r, int64(max)+1))
	if e != nil {
		return nil, e
	}
	if len(data) > max {
		return nil, errors.New("Input exceeds the supported size limit.")
	}
	return data, nil
}
func selfTest(dir string) error {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	password := "Synthetic independent passphrase!"
	files := []vaultFile{{Path: "demo/first.txt", Size: 16, Type: "text/plain", Data: []byte("Synthetic sample")}, {Path: "demo/nested/empty.bin", Data: []byte{}}}
	encrypted, e := sealAES(files, password)
	if e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(dir, "native-cs4.cen"), encrypted, 0600); e != nil {
		return e
	}
	restored, e := openPackage(encrypted, password)
	if e != nil {
		return e
	}
	if len(restored) != 2 || !bytes.Equal(restored[0].Data, files[0].Data) {
		return errors.New("AES round trip failed")
	}
	changed := bytes.Clone(encrypted)
	changed[len(changed)-1] ^= 1
	if _, e = openPackage(changed, password); e == nil {
		return errors.New("altered package accepted")
	}
	if _, e = openPackage(encrypted, "wrong password"); e == nil {
		return errors.New("wrong password accepted")
	}
	x, e := sealX(payload{Name: "note.txt", Text: true, Data: []byte("Synthetic XChaCha note")}, password)
	if e != nil {
		return e
	}
	opened, e := openPackage(x, password)
	if e != nil || string(opened[0].Data) != "Synthetic XChaCha note" {
		return errors.New("XChaCha round trip failed")
	}
	folder, e := restoreNewFolder(dir, restored)
	if e != nil {
		return e
	}
	data, e := os.ReadFile(filepath.Join(folder, "demo", "first.txt"))
	if e != nil || string(data) != "Synthetic sample" {
		return errors.New("folder restore failed")
	}
	report := map[string]any{"aes_round_trip": true, "xchacha_round_trip": true, "tamper_rejected": true, "wrong_password_rejected": true, "nested_folder_restore": true, "browser_required": false}
	b, _ := json.MarshalIndent(report, "", "  ")
	return os.WriteFile(filepath.Join(dir, "self-test.json"), b, 0600)
}
