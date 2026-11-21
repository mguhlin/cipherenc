// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tk "modernc.org/tk9.0"
)

const navy = "#0a1d3a"
const blue = "#1769e8"
const silver = "#d8e2ee"
const paper = "#f8fbff"

type nativeUI struct {
	showing                                  bool
	action, source, algorithm                *tk.TComboboxWidget
	password, confirm                        *tk.EntryWidget
	note                                     *tk.TextWidget
	selected, status                         *tk.LabelWidget
	choose, run, save, clear, generate, show *tk.ButtonWidget
	path                                     string
	busy                                     bool
	output                                   []byte
	files                                    []vaultFile
	resultName                               string
	lastActivity                             time.Time
}

func (u *nativeUI) clearWork() {
	clear(u.output)
	u.output = nil
	for i := range u.files {
		clear(u.files[i].Data)
	}
	u.files = nil
	u.path = ""
	u.selected.Configure(tk.Txt("No file or folder selected"))
	u.password.Configure(tk.Textvariable(""), tk.Show("•"))
	u.showing = false
	u.confirm.Configure(tk.Textvariable(""))
	u.note.Delete("1.0", "end")
	u.save.Configure(tk.State("disabled"))
	u.status.Configure(tk.Txt("Workspace cleared. Your saved files and originals remain unchanged."))
}
func (u *nativeUI) lock(value bool) {
	u.busy = value
	state := "normal"
	if value {
		state = "disabled"
	}
	for _, b := range []*tk.ButtonWidget{u.choose, u.run, u.clear, u.generate, u.show, u.save} {
		b.Configure(tk.State(state))
	}
	u.password.Configure(tk.State(state))
	u.confirm.Configure(tk.State(state))
	u.note.Configure(tk.State(state))
	combo := "readonly"
	if value {
		combo = "disabled"
	}
	u.action.Configure(tk.State(combo))
	u.source.Configure(tk.State(combo))
	u.algorithm.Configure(tk.State(combo))
	if !value && len(u.output) == 0 && len(u.files) == 0 {
		u.save.Configure(tk.State("disabled"))
	}
}
func (u *nativeUI) chooseInput() {
	if u.busy {
		return
	}
	if u.source.Textvariable() == "Folder" && u.action.Textvariable() == "Encrypt" {
		u.path = tk.ChooseDirectory(tk.Title("Choose folder to encrypt"))
	} else {
		selected := tk.GetOpenFile(tk.Title("Choose file or encrypted package"))
		if len(selected) > 0 {
			u.path = selected[0]
		}
	}
	if u.path != "" {
		u.selected.Configure(tk.Txt(u.path))
	}
	u.lastActivity = time.Now()
}
func (u *nativeUI) begin() {
	if u.busy {
		return
	}
	action := u.action.Textvariable()
	password := u.password.Textvariable()
	algorithm := u.algorithm.Textvariable()
	source := u.source.Textvariable()
	note := u.note.Text()
	input := u.path
	if password == "" {
		u.status.Configure(tk.Txt("Enter your password first."))
		tk.Focus(u.password)
		return
	}
	if action == "Encrypt" && (len([]rune(password)) < 12 || password != u.confirm.Textvariable()) {
		u.status.Configure(tk.Txt("Use at least 12 characters and enter the same password twice."))
		tk.Focus(u.password)
		return
	}
	if source != "Text" && input == "" {
		u.status.Configure(tk.Txt("Choose a file or folder first."))
		return
	}
	if action != "Encrypt" && source == "Text" && !strings.HasPrefix(strings.TrimSpace(note), "PRISMVAULT") {
		u.status.Configure(tk.Txt("To decrypt a saved .cen package, select File and choose the package. Older PRISMVAULT text can be pasted here."))
		return
	}
	clear(u.output)
	u.output = nil
	for i := range u.files {
		clear(u.files[i].Data)
	}
	u.files = nil
	u.lock(true)
	u.status.Configure(tk.Txt("Working on your device. Hardening the password and authenticating your data…"))
	u.lastActivity = time.Now()
	go func() {
		var output []byte
		var files []vaultFile
		var err error
		name := "encrypted.cen"
		if action == "Encrypt" {
			if source == "Text" {
				data := []byte(note)
				if len(data) > nativeLimit {
					err = fmt.Errorf("Text exceeds 256 MiB.")
				} else {
					files = []vaultFile{{Path: "private-note.txt", Size: len(data), Type: "text/plain", Data: data}}
				}
			} else {
				files, err = collectFiles(input)
				name = filepath.Base(input) + ".cen"
			}
			if err == nil && algorithm == "XChaCha20-Poly1305" {
				if len(files) != 1 || strings.Contains(files[0].Path, "/") {
					err = fmt.Errorf("XChaCha20 handles one file or note up to 16 MiB. Choose AES-256-GCM for folders.")
				} else {
					output, err = sealX(payload{Name: filepath.Base(files[0].Path), Text: source == "Text", Data: files[0].Data}, password)
				}
			} else if err == nil {
				output, err = sealAES(files, password)
			}
			for i := range files {
				clear(files[i].Data)
			}
			files = nil
		} else {
			var data []byte
			if source == "Text" {
				data = []byte(note)
			} else {
				var file *os.File
				file, err = os.Open(input)
				if err == nil {
					st, e := file.Stat()
					if e != nil {
						err = e
					} else if st.Size() > nativeLimit+(16<<20) {
						err = fmt.Errorf("Package exceeds the 272 MiB import limit.")
					} else {
						data, err = readLimited(file, nativeLimit+(16<<20))
					}
					file.Close()
				}
			}
			if err == nil {
				files, err = openPackage(data, password)
			}
			clear(data)
			if action == "Verify" {
				for i := range files {
					clear(files[i].Data)
				}
				files = nil
			}
		}
		tk.PostEvent(func() {
			u.output = output
			u.files = files
			u.resultName = name
			u.lock(false)
			if err != nil {
				u.status.Configure(tk.Txt("Stopped: " + err.Error()))
				return
			}
			u.password.Configure(tk.Textvariable(""), tk.Show("•"))
			u.showing = false
			u.confirm.Configure(tk.Textvariable(""))
			u.note.Delete("1.0", "end")
			u.path = ""
			u.selected.Configure(tk.Txt("Input released from the workspace"))
			message := "Encryption complete. Save the .cen package, then test decryption before deleting an original."
			if action == "Decrypt" {
				message = fmt.Sprintf("Authenticated %d file(s). Save the restored result below.", len(files))
			}
			if action == "Verify" {
				message = "Every package record authenticated. Restored plaintext was discarded. This does not identify the sender."
			}
			u.status.Configure(tk.Txt(message))
			u.lastActivity = time.Now()
		}, false)
	}()
}
func (u *nativeUI) saveResult() {
	if u.busy {
		return
	}
	// Keep the result alive throughout modal picker loops and the write itself.
	// Inactivity clearing and other operations must not mutate a pending save.
	u.lock(true)
	defer u.lock(false)
	var err error
	var destination string
	if len(u.output) > 0 {
		destination = tk.GetSaveFile(tk.Title("Save encrypted package — choose a new filename"), tk.Initialfile(u.resultName), tk.Defaultextension(".cen"))
		if destination != "" {
			if !strings.EqualFold(filepath.Ext(destination), ".cen") {
				destination += ".cen"
			}
			err = saveNew(destination, u.output)
		}
	} else if len(u.files) == 1 && !strings.Contains(u.files[0].Path, "/") {
		destination = tk.GetSaveFile(tk.Title("Save restored file — choose a new filename"), tk.Initialfile(u.files[0].Path))
		if destination != "" {
			err = saveNew(destination, u.files[0].Data)
		}
	} else if len(u.files) > 0 {
		parent := tk.ChooseDirectory(tk.Title("Choose where to create a new restoration folder"))
		if parent != "" {
			destination, err = restoreNewFolder(parent, u.files)
		}
	}
	if err != nil {
		u.status.Configure(tk.Txt("Save stopped: " + err.Error()))
		return
	}
	if destination != "" {
		u.status.Configure(tk.Txt("Saved: " + destination + "\nKeep your password safe. Existing files were not replaced."))
	}
	u.lastActivity = time.Now()
}
func runGUI(smokeDir string) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("Native window could not open: %v", p)
		}
	}()
	tk.App.WmTitle("CipherEnc — Offline encryption")
	tk.App.Configure(tk.Background(paper))
	tk.WmGeometry(tk.App, "980x820")
	tk.WmMinSize(tk.App, 720, 700)
	font := tk.Font("Helvetica", 11)
	tk.FontConfigure("TkDefaultFont", tk.Family("Helvetica"), tk.Size(11))
	frame := tk.Frame(tk.Background(paper), tk.Padx(24), tk.Pady(20))
	tk.Pack(frame, tk.Fill("both"), tk.Expand(true))
	tk.GridColumnConfigure(frame, 0, tk.Weight(1))
	tk.GridColumnConfigure(frame, 1, tk.Weight(1))
	label := func(text string) *tk.LabelWidget {
		return frame.Label(tk.Txt(text), font, tk.Background(paper), tk.Foreground(navy), tk.Anchor("w"), tk.Justify("left"))
	}
	tk.Grid(frame.Label(tk.Txt("CipherEnc"), tk.Font("Helvetica", 27, "bold"), tk.Background(paper), tk.Foreground(navy)), tk.Row(0), tk.Column(0), tk.Sticky("w"))
	tk.Grid(frame.Label(tk.Txt("OFFLINE · NATIVE DESKTOP"), font, tk.Background(paper), tk.Foreground("#225581")), tk.Row(0), tk.Column(1), tk.Sticky("e"))
	tk.Grid(label("Protect files, folders, and text on this device. Encrypted files end in .cen."), tk.Row(1), tk.Columnspan(2), tk.Sticky("w"), tk.Pady(10))
	u := &nativeUI{lastActivity: time.Now()}
	u.action = frame.TCombobox(tk.Values([]string{"Encrypt", "Decrypt", "Verify"}), tk.Textvariable("Encrypt"), tk.State("readonly"), tk.Width(16), font)
	u.source = frame.TCombobox(tk.Values([]string{"File", "Folder", "Text"}), tk.Textvariable("File"), tk.State("readonly"), tk.Width(16), font)
	u.algorithm = frame.TCombobox(tk.Values([]string{"AES-256-GCM", "XChaCha20-Poly1305"}), tk.Textvariable("AES-256-GCM"), tk.State("readonly"), tk.Width(26), font)
	tk.Grid(label("Action"), tk.Row(2), tk.Column(0), tk.Sticky("w"))
	tk.Grid(label("Data type"), tk.Row(2), tk.Column(1), tk.Sticky("w"))
	tk.Grid(u.action, tk.Row(3), tk.Column(0), tk.Sticky("ew"), tk.Padx(4), tk.Pady(6))
	tk.Grid(u.source, tk.Row(3), tk.Column(1), tk.Sticky("ew"), tk.Padx(4), tk.Pady(6))
	tk.Grid(label("Encryption algorithm · detected automatically when decrypting"), tk.Row(4), tk.Columnspan(2), tk.Sticky("w"), tk.Pady(5))
	tk.Grid(u.algorithm, tk.Row(5), tk.Columnspan(2), tk.Sticky("ew"), tk.Padx(4), tk.Pady(4))
	button := func(text string, fn func()) *tk.ButtonWidget {
		return frame.Button(tk.Txt(text), tk.Command(fn), font, tk.Background("white"), tk.Foreground(navy), tk.Activebackground(silver), tk.Padx(12), tk.Pady(9), tk.Relief("solid"), tk.Borderwidth(1))
	}
	u.choose = button("Choose file / folder…", u.chooseInput)
	tk.Grid(u.choose, tk.Row(6), tk.Column(0), tk.Sticky("w"), tk.Pady(8))
	u.selected = label("No file or folder selected")
	u.selected.Configure(tk.Wraplength(850))
	tk.Grid(u.selected, tk.Row(7), tk.Columnspan(2), tk.Sticky("ew"))
	tk.Grid(label("Text to encrypt · or paste an older PRISMVAULT text package to decrypt"), tk.Row(8), tk.Columnspan(2), tk.Sticky("w"), tk.Pady(6))
	u.note = frame.Text(font, tk.Height(5), tk.Width(60), tk.Wrap("word"), tk.Background("white"), tk.Foreground(navy), tk.Padx(8), tk.Pady(8), tk.Undo(false))
	tk.Grid(u.note, tk.Row(9), tk.Columnspan(2), tk.Sticky("nsew"))
	tk.GridRowConfigure(frame, 9, tk.Weight(1))
	tk.Grid(label("Password · 12+ characters when encrypting"), tk.Row(10), tk.Column(0), tk.Sticky("w"), tk.Pady(7))
	tk.Grid(label("Confirm password · encryption only"), tk.Row(10), tk.Column(1), tk.Sticky("w"))
	u.password = frame.Entry(font, tk.Textvariable(""), tk.Show("•"), tk.Background("white"), tk.Foreground(navy))
	u.confirm = frame.Entry(font, tk.Textvariable(""), tk.Show("•"), tk.Background("white"), tk.Foreground(navy))
	tk.Grid(u.password, tk.Row(11), tk.Column(0), tk.Sticky("ew"), tk.Padx(4), tk.Ipady(6))
	tk.Grid(u.confirm, tk.Row(11), tk.Column(1), tk.Sticky("ew"), tk.Padx(4), tk.Ipady(6))
	u.generate = button("Generate strong password", func() {
		const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%&*+-="
		bytes := make([]byte, 32)
		if _, e := rand.Read(bytes); e != nil {
			u.status.Configure(tk.Txt("Secure randomness is unavailable."))
			return
		}
		var value strings.Builder
		for _, b := range bytes {
			if int(b) >= 256-(256%len(alphabet)) {
				continue
			}
			value.WriteByte(alphabet[int(b)%len(alphabet)])
		}
		if value.Len() < 24 {
			u.status.Configure(tk.Txt("Please generate again."))
			return
		}
		u.password.Configure(tk.Textvariable(value.String()), tk.Show(""))
		u.showing = true
		u.confirm.Configure(tk.Textvariable(value.String()))
		clear(bytes)
		u.status.Configure(tk.Txt("A random password was generated. Record it securely before encrypting; it cannot be recovered."))
	})
	u.show = button("Show / hide password", func() {
		show := ""
		if u.showing {
			show = "•"
		}
		u.showing = !u.showing
		u.password.Configure(tk.Show(show))
	})
	tk.Grid(u.generate, tk.Row(12), tk.Column(0), tk.Sticky("w"), tk.Pady(10))
	tk.Grid(u.show, tk.Row(12), tk.Column(1), tk.Sticky("w"))
	u.run = button("Run selected action", u.begin)
	u.run.Configure(tk.Background(blue), tk.Foreground("white"))
	u.save = button("Save result…", u.saveResult)
	u.save.Configure(tk.State("disabled"))
	tk.Grid(u.run, tk.Row(13), tk.Column(0), tk.Sticky("ew"), tk.Padx(4))
	tk.Grid(u.save, tk.Row(13), tk.Column(1), tk.Sticky("ew"), tk.Padx(4))
	u.status = label("Ready. AES-256-GCM is the default. Nothing is uploaded; no browser is needed.")
	u.status.Configure(tk.Background("#eaf2fc"), tk.Padx(12), tk.Pady(12), tk.Wraplength(850))
	tk.Grid(u.status, tk.Row(14), tk.Columnspan(2), tk.Sticky("ew"), tk.Pady(12))
	u.clear = button("Clear workspace", func() {
		if !u.busy {
			u.clearWork()
		}
	})
	tk.Grid(u.clear, tk.Row(15), tk.Column(0), tk.Sticky("w"))
	quit := button("Quit CipherEnc", func() {
		if !u.busy {
			u.clearWork()
			tk.Destroy(tk.App)
		}
	})
	tk.Grid(quit, tk.Row(15), tk.Column(1), tk.Sticky("e"))
	tk.Grid(label("AES: files/folders/text up to 256 MiB. XChaCha20: one file/text up to 16 MiB.\nPasswords cannot be recovered. Test decryption before deleting originals."), tk.Row(16), tk.Columnspan(2), tk.Sticky("w"), tk.Pady(10))
	tk.WmProtocol(tk.App, "WM_DELETE_WINDOW", tk.Command(func() {
		if !u.busy {
			u.clearWork()
			tk.Destroy(tk.App)
		}
	}))
	tk.Bind(tk.App, "<KeyPress>", tk.Command(func() { u.lastActivity = time.Now() }))
	tk.Bind(tk.App, "<ButtonPress>", tk.Command(func() { u.lastActivity = time.Now() }))
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				tk.PostEvent(func() {
					if !u.busy && time.Since(u.lastActivity) > 10*time.Minute {
						u.clearWork()
					}
				}, false)
			}
		}
	}()

	if smokeDir != "" {
		go func() {
			time.Sleep(1200 * time.Millisecond)
			tk.PostEvent(func() {
				u.source.Configure(tk.Textvariable("Text"))
				u.note.Insert("1.0", "A private lesson plan stays on this device.\n\n")
				u.password.Configure(tk.Textvariable("Synthetic UI smoke passphrase!"))
				u.confirm.Configure(tk.Textvariable("Synthetic UI smoke passphrase!"))
			}, false)
			time.Sleep(1800 * time.Millisecond)
			tk.PostEvent(func() { u.begin() }, false)
			time.Sleep(1800 * time.Millisecond)
			tk.PostEvent(func() {
				_ = os.MkdirAll(smokeDir, 0700)
				report := map[string]any{"native_window": true, "title": tk.App.WmTitle(""), "browser_required": false, "ui_encryption": len(u.output) > 0}
				if len(u.output) > 0 {
					_ = os.WriteFile(filepath.Join(smokeDir, "native-ui.cen"), u.output, 0600)
					restored, e := openPackage(u.output, "Synthetic UI smoke passphrase!")
					report["trailing_newline_round_trip"] = e == nil && len(restored) == 1 && string(restored[0].Data) == "A private lesson plan stays on this device.\n\n"
					if report["trailing_newline_round_trip"] != true {
						err = fmt.Errorf("Native UI text round trip failed.")
					}
				}
				b, _ := json.MarshalIndent(report, "", "  ")
				_ = os.WriteFile(filepath.Join(smokeDir, "native-ui.json"), b, 0600)
			}, false)
			time.Sleep(4 * time.Second)
			tk.PostEvent(func() { u.clearWork(); tk.Destroy(tk.App) }, false)
		}()
	}

	tk.App.Center().Wait()
	return err
}
