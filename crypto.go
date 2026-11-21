// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const maxData = 16 << 20
const maxPackage = 24 << 20
const xMagic = "CENCX01!"

var errUnlock = errors.New("Unlock failed. The password is wrong or the package was altered.")

type payload struct {
	Name string `json:"name"`
	Text bool   `json:"text"`
	Data []byte `json:"data"`
}
type cryptoInput struct {
	Operation string `json:"operation"`
	Password  string `json:"password"`
	payload
}

func validPayload(p payload) bool {
	return len(p.Data) <= maxData && len(p.Name) > 0 && len(p.Name) <= 240 && utf8.ValidString(p.Name) && !strings.ContainsAny(p.Name, "/\\\x00\r\n") && p.Name != "." && p.Name != ".."
}

// CENCX01! | salt[16] | nonce[24] | ciphertext and Poly1305 tag.
// The exact 48-byte header is AAD. KDF settings are fixed by this format version,
// so imported headers cannot request unbounded memory or CPU work.
func sealX(p payload, password string) ([]byte, error) {
	if !validPayload(p) {
		return nil, errors.New("Use a valid filename and at most 16 MiB of data.")
	}
	if utf8.RuneCountInString(password) < 12 || len(password) > 4096 {
		return nil, errors.New("Use a password of at least 12 characters (maximum 4096 UTF-8 bytes).")
	}
	plain, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	defer clear(plain)
	header := make([]byte, 48)
	copy(header, xMagic)
	if _, err = rand.Read(header[8:]); err != nil {
		return nil, err
	}
	key := argon2.IDKey([]byte(password), header[8:24], 3, 65536, 1, 32)
	defer clear(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	return aead.Seal(header, header[24:48], plain, header), nil
}

func openX(data []byte, password string) (payload, error) {
	var p payload
	if len(data) < 64 || len(data) > maxPackage || string(data[:8]) != xMagic {
		return p, errors.New("This is not a supported CipherEnc XChaCha20 package.")
	}
	if len(password) == 0 || len(password) > 4096 {
		return p, errUnlock
	}
	key := argon2.IDKey([]byte(password), data[8:24], 3, 65536, 1, 32)
	defer clear(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return p, err
	}
	plain, err := aead.Open(nil, data[24:48], data[48:], data[:48])
	if err != nil {
		return p, errUnlock
	}
	defer clear(plain)
	if json.Unmarshal(plain, &p) != nil || !validPayload(p) {
		return payload{}, errors.New("Invalid authenticated package contents.")
	}
	return p, nil
}
