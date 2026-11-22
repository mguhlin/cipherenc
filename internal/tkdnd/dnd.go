// SPDX-License-Identifier: AGPL-3.0-or-later
// Native TkDND is bundled for each desktop target; nothing is downloaded at run time.
package tkdnd

import (
	"archive/zip"
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	tk "modernc.org/tk9.0"
)

//go:embed assets/*.zip
var assets embed.FS

type Bridge struct {
	Context tk.ExtensionContext
	dir     string
	Loaded  bool
}

func (b *Bridge) Initialize(c tk.ExtensionContext) error {
	b.Context = c
	data, err := assets.ReadFile("assets/" + runtime.GOOS + "-" + runtime.GOARCH + ".zip")
	if err != nil {
		return err
	}
	b.dir, err = os.MkdirTemp("", "cipherenc-dnd-")
	if err != nil {
		return err
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range z.File {
		if filepath.Base(f.Name) != f.Name {
			return fmt.Errorf("Invalid bundled extension path")
		}
		r, e := f.Open()
		if e != nil {
			return e
		}
		content, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			return e
		}
		if e = os.WriteFile(filepath.Join(b.dir, f.Name), content, 0600); e != nil {
			return e
		}
	}
	if _, err = c.Evalv("lappend", "auto_path", b.dir); err != nil {
		return err
	}
	_, err = c.Evalv("package", "require", "tkdnd")
	b.Loaded = err == nil

	return err
}
func (b *Bridge) Close() {
	if b.dir != "" {
		_ = os.RemoveAll(b.dir)
	}
}

// Bind uses TkDND's list-quoted %D substitution, then parses it as a Tcl list.
// Paths are never interpreted as Tcl code, split on spaces, or passed to a shell.
func (b *Bridge) Bind(target *tk.LabelWidget, receive func([]string)) error {
	c := b.Context
	callback := tk.Button(tk.Command(func() {
		value, err := c.Evalv("set", "::cipherenc_drop_paths")
		if err == nil {
			receive(c.ParseList(value))
		}
	}))
	if _, err := c.Evalv("tkdnd::drop_target", "register", target.String(), "DND_Files"); err != nil {
		return err
	}
	script := "set ::cipherenc_drop_paths %D; after idle {" + callback.String() + " invoke}; list copy"
	// Some platform mappings return several independent type names, so retain
	// the generic binding too. This target registers only file-list drops.
	if _, err := c.Evalv("bind", target.String(), "<<Drop>>", script); err != nil {
		return err
	}
	_, err := c.Evalv("bind", target.String(), "<<Drop:DND_Files>>", script)
	return err
}
