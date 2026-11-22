// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mguhlin/cipherenc/internal/tkdnd"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	tk "modernc.org/tk9.0"
)

const navy = "#0a1d3a"
const blue = "#1769e8"
const silver = "#d8e2ee"
const paper = "#f8fbff"

type nativeUI struct {
	quick                                    *quickUI
	smokeDir                                 string
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
	if u.quick != nil {
		u.quick.cancelAutoStart()
		u.quick.paths = nil
		u.quick.drop.Configure(tk.Txt("Drop files or folders here"))
		u.quick.current.Configure(tk.Value(0))
		u.quick.all.Configure(tk.Value(0))
	}
	u.selected.Configure(tk.Txt("No file or folder selected"))
	u.password.Configure(tk.Textvariable(""), tk.Show("•"))
	u.showing = false
	u.confirm.Configure(tk.Textvariable(""), tk.Show("•"))
	u.note.Delete("1.0", "end")
	u.save.Configure(tk.State("disabled"))
	tk.GridRemove(u.save.Window)
	u.status.Configure(tk.Txt("Workspace cleared. Your saved files and originals remain unchanged."))
}
func (u *nativeUI) lock(value bool) {
	u.busy = value
	if u.quick != nil {
		u.quick.lock(value)
	}
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
				if source == "Text" {
					output, err = sealText(note, password)
				} else {
					output, err = sealAES(files, password)
				}
			}
			if err == nil {
				var checked []vaultFile
				checked, err = openPackage(output, password)
				if err == nil && (len(checked) != 1 || string(checked[0].Data) != note) {
					err = fmt.Errorf("Encryption verification failed")
				}
				for i := range checked {
					clear(checked[i].Data)
				}
			}
			for i := range files {
				clear(files[i].Data)
			}
			files = nil
		} else {
			var data []byte
			if source == "Text" {
				data = []byte(strings.TrimSpace(note))
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
			u.confirm.Configure(tk.Textvariable(""), tk.Show("•"))
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
			if source == "Text" {
				if action == "Decrypt" && len(files) == 1 && utf8.Valid(files[0].Data) {
					u.note.Insert("1.0", string(files[0].Data))
					u.status.Configure(tk.Txt("Text unlocked. Every encrypted record authenticated. You can select and copy the text above."))
				} else {
					if algorithm == "AES-256-GCM" {
						u.note.Insert("1.0", string(output))
					}
					if u.smokeDir != "" {
						err = saveNew(filepath.Join(u.smokeDir, "native-note.cen"), output)
					} else {
						u.saveResult()
					}
				}
			}
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
		u.status.Configure(tk.Txt("Save stopped: " + err.Error() + ". Your result is retained; choose Save result to try again."))
		tk.Grid(u.save)
		return
	}
	if destination != "" {
		u.status.Configure(tk.Txt("Saved: " + destination + "\nKeep your password safe. Existing files were not replaced."))
		tk.GridRemove(u.save.Window)
	} else {
		u.status.Configure(tk.Txt("Save cancelled. Your encrypted result is retained; choose Save result to try again."))
		tk.Grid(u.save)
	}
	u.lastActivity = time.Now()
}

type quickUI struct {
	u                                  *nativeUI
	paths                              []string
	drop                               *tk.LabelWidget
	current, all                       *tk.TProgressbarWidget
	files, folder, start, stop, toggle *tk.ButtonWidget
	cancelled                          atomic.Bool
	silent                             bool
	completed                          bool
	autoStart                          string
	decryptOnly                        bool
}

func (q *quickUI) cancelAutoStart() {
	if q.autoStart != "" {
		tk.TclAfterCancel(q.autoStart)
		q.autoStart = ""
	}
}

// A matching confirmation marks a complete encryption password. Wait briefly
// for typing/paste bindings to finish, then recheck live state on the UI thread.
// Decryption passwords have no confirmation: Enter explicitly ends their input.
func (q *quickUI) credentialsChanged() {
	q.cancelAutoStart()
	if q.u.busy || len(q.paths) == 0 || q.decryptOnly {
		return
	}
	q.autoStart = tk.TclAfter(400*time.Millisecond, func() {
		q.autoStart = ""
		if !q.u.busy && len(q.paths) > 0 && !q.decryptOnly {
			password := q.u.password.Textvariable()
			if len([]rune(password)) >= 12 && password == q.u.confirm.Textvariable() {
				q.begin()
			}
		}
	})
}

func (q *quickUI) lock(busy bool) {
	if busy {
		q.cancelAutoStart()
	}
	state := "normal"
	if busy {
		state = "disabled"
	}
	for _, b := range []*tk.ButtonWidget{q.files, q.folder, q.start, q.toggle} {
		b.Configure(tk.State(state))
	}
	stop := "disabled"
	if busy {
		stop = "normal"
	}
	q.stop.Configure(tk.State(stop))
}
func (q *quickUI) accept(paths []string) {
	if q.u.busy || len(paths) == 0 {
		return
	}
	q.cancelAutoStart()
	if len(paths) > 100 {
		q.u.status.Configure(tk.Txt("Choose at most 100 items per batch."))
		return
	}
	q.paths = nil
	q.decryptOnly = true
	for _, path := range paths {
		encrypted, err := isEncryptedInput(path)
		if err != nil {
			q.u.status.Configure(tk.Txt("Selection stopped: " + err.Error()))
			return
		}
		q.decryptOnly = q.decryptOnly && encrypted
	}
	seen := map[string]bool{}
	q.paths = nil
	for _, p := range paths {
		abs, e := filepath.Abs(filepath.Clean(p))
		if e != nil {
			q.u.status.Configure(tk.Txt(e.Error()))
			return
		}
		if !seen[abs] {
			q.paths = append(q.paths, abs)
			seen[abs] = true
		}
	}
	// A selected folder already contains its selected descendants. Avoid making
	// an extra archive inside that folder before the folder itself is archived.
	var roots []string
	for i, path := range q.paths {
		covered := false
		for j, parent := range q.paths {
			if i == j {
				continue
			}
			st, e := os.Lstat(parent)
			if e != nil || !st.IsDir() {
				continue
			}
			rel, e := filepath.Rel(parent, path)
			if e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				covered = true
				break
			}
		}
		if !covered {
			roots = append(roots, path)
		}
	}
	q.paths = roots
	q.u.note.Delete("1.0", "end")
	q.drop.Configure(tk.Txt(fmt.Sprintf("%d item(s) selected\n%s", len(q.paths), filepath.Base(q.paths[0]))))
	q.u.lastActivity = time.Now()
	password := q.u.password.Textvariable()
	if password != "" && (onlyEncrypted(q.paths) || len([]rune(password)) >= 12 && password == q.u.confirm.Textvariable()) {
		q.begin()
		return
	}
	if q.decryptOnly {
		q.u.status.Configure(tk.Txt("Encrypted selection ready. Enter its password and press Enter to unlock beside the original."))
	} else {
		q.u.status.Configure(tk.Txt("Selection ready. Enter and confirm a 12+ character password. Encryption starts automatically and saves .cen beside each original."))
	}
	tk.Focus(q.u.password)
}
func (q *quickUI) begin() {
	q.cancelAutoStart()
	u := q.u
	if !u.busy && len(q.paths) == 0 && u.note.Text() != "" {
		u.source.Configure(tk.Textvariable("Text"))
		action := "Encrypt"
		if strings.HasPrefix(strings.TrimSpace(u.note.Text()), "PRISMVAULT") {
			action = "Decrypt"
		}
		u.action.Configure(tk.Textvariable(action))
		u.begin()
		return
	}
	if u.busy {
		return
	}
	password := u.password.Textvariable()
	if password == "" || !onlyEncrypted(q.paths) && (len([]rune(password)) < 12 || password != u.confirm.Textvariable()) {
		u.status.Configure(tk.Txt("Enter a password. To encrypt, use 12+ characters and enter the same password twice."))
		tk.Focus(u.password)
		return
	}
	if len(q.paths) == 0 {
		u.status.Configure(tk.Txt("Drop or choose your files or folders first."))
		return
	}
	paths := append([]string(nil), q.paths...)
	q.cancelled.Store(false)
	q.completed = false
	u.lock(true)
	q.current.Configure(tk.Value(0))
	q.all.Configure(tk.Value(0))
	u.status.Configure(tk.Txt("Working locally. Originals are kept unless you explicitly choose to shred them."))
	go func() {
		var results []string
		var workErr error
		check := func() error {
			if q.cancelled.Load() {
				return errors.New("Stopped. Completed archives remain saved; originals were kept unless you already chose shredding.")
			}
			return nil
		}
		for index, path := range paths {
			if workErr = check(); workErr != nil {
				break
			}
			tk.PostEvent(func() {
				q.drop.Configure(tk.Txt(fmt.Sprintf("%d of %d\n%s", index+1, len(paths), filepath.Base(path))))
				q.current.Configure(tk.Value(0))
			}, false)
			progress := func(value float64) {
				tk.PostEvent(func() {
					q.current.Configure(tk.Value(value))
					q.all.Configure(tk.Value((float64(index) + value/100) / float64(len(paths)) * 100))
				}, false)
			}
			decrypt, detectErr := isEncryptedInput(path)
			if detectErr != nil {
				workErr = detectErr
				break
			}
			if decrypt {
				var destination string
				destination, workErr = decryptAdjacent(path, password)
				if workErr != nil {
					break
				}
				results = append(results, "Restored: "+destination)
				progress(100)
			} else {
				var p *archivePlan
				p, workErr = encryptAdjacent(path, password, check, progress)
				if workErr != nil {
					break
				}
				shred := false
				if !q.silent && !q.cancelled.Load() {
					answer := make(chan bool, 1)
					tk.PostEvent(func() {
						// A modal loop processes events, so the workspace stays locked throughout.
						q.stop.Configure(tk.State("disabled"))
						choice := tk.MessageBox(tk.Type("yesno"), tk.Default("no"), tk.Icon("warning"), tk.Title("Shred the original?"), tk.Msg("Encrypted archive saved and verified.\n\n"+p.destination+"\n\nShred the original "+filepath.Base(p.source)+"?"), tk.Detail("No keeps the original (recommended). Yes overwrites accessible file bytes once, then removes the original file or folder. This cannot be undone. SSDs, backups, cloud copies and some filesystems may retain data. Close any editors using the original first. Your password cannot be recovered."))
						answer <- choice == "yes"
					}, false)
					shred = <-answer
				}
				outcome := "Original kept."
				if shred {
					tk.PostEvent(func() {
						u.status.Configure(tk.Txt("Shredding the confirmed original. Keep the verified .cen archive and its password."))
					}, false)
					workErr = p.shred(password)
					outcome = "Original shredded (best effort)."
					if workErr != nil {
						workErr = fmt.Errorf("Verified archive: %s\nShredding stopped; some originals may remain or have been overwritten: %w", p.destination, workErr)
					}
				}
				results = append(results, "Saved and verified: "+p.destination+"\n"+outcome)
				p.close()
				tk.PostEvent(func() { q.stop.Configure(tk.State("normal")) }, false)
				if workErr != nil {
					break
				}
			}
		}
		tk.PostEvent(func() {
			u.lock(false)
			q.paths = nil
			q.completed = workErr == nil
			u.password.Configure(tk.Textvariable(""), tk.Show("•"))
			u.confirm.Configure(tk.Textvariable(""), tk.Show("•"))
			u.showing = false
			msg := strings.Join(results, "\n\n")
			if len(results) > 2 {
				msg = fmt.Sprintf("Completed %d items.\n%s", len(results), results[len(results)-1])
			}
			if workErr != nil {
				msg += "\nStopped: " + workErr.Error()
			}
			if msg == "" {
				msg = "No files processed."
			}
			u.status.Configure(tk.Txt(msg))
			q.drop.Configure(tk.Txt("Drop more files or folders here"))
			u.lastActivity = time.Now()
		}, false)
	}()
}

func runGUI(smokeDir string) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("Native window could not open: %v", p)
		}
	}()
	tk.App.WmTitle("CipherEnc " + version + " — Offline encryption")
	tk.App.Configure(tk.Background(paper))
	tk.WmGeometry(tk.App, "880x760")
	tk.WmMinSize(tk.App, 760, 720)
	tk.FontConfigure("TkDefaultFont", tk.Family("Helvetica"), tk.Size(11))
	font := tk.Font("Helvetica", 11)
	frame := tk.Frame(tk.Background(paper), tk.Padx(24), tk.Pady(18))
	tk.Pack(frame, tk.Fill("both"), tk.Expand(true))
	tk.GridColumnConfigure(frame, 0, tk.Weight(1))
	tk.GridColumnConfigure(frame, 1, tk.Weight(1))
	tk.GridRowConfigure(frame, 6, tk.Weight(1))
	label := func(parent *tk.Window, text string) *tk.LabelWidget {
		return parent.Label(tk.Txt(text), font, tk.Background(paper), tk.Foreground(navy), tk.Anchor("w"), tk.Justify("left"))
	}
	button := func(parent *tk.Window, text string, fn func()) *tk.ButtonWidget {
		return parent.Button(tk.Txt(text), tk.Command(fn), font, tk.Background("white"), tk.Foreground(navy), tk.Activebackground(silver), tk.Padx(12), tk.Pady(8), tk.Relief("solid"), tk.Borderwidth(1))
	}
	tk.Grid(frame.Label(tk.Txt("CipherEnc"), tk.Font("Helvetica", 26, "bold"), tk.Background(paper), tk.Foreground(navy)), tk.Row(0), tk.Column(0), tk.Sticky("w"))
	tk.Grid(label(frame.Window, "v"+version+" · Offline. On your device."), tk.Row(0), tk.Column(1), tk.Sticky("e"))
	tk.Grid(label(frame.Window, "Drop it. Lock it. Keep it beside the original."), tk.Row(1), tk.Columnspan(2), tk.Sticky("w"), tk.Pady(10))
	u := &nativeUI{lastActivity: time.Now(), smokeDir: smokeDir}
	q := &quickUI{u: u, silent: smokeDir != ""}
	u.quick = q
	tk.Grid(label(frame.Window, "Password · 12+ characters to encrypt"), tk.Row(2), tk.Column(0), tk.Sticky("w"), tk.Pady(5))
	tk.Grid(label(frame.Window, "Confirm password · encryption only"), tk.Row(2), tk.Column(1), tk.Sticky("w"))
	u.password = frame.Entry(font, tk.Textvariable(""), tk.Show("•"), tk.Background("white"), tk.Foreground(navy))
	u.confirm = frame.Entry(font, tk.Textvariable(""), tk.Show("•"), tk.Background("white"), tk.Foreground(navy))
	tk.Grid(u.password, tk.Row(3), tk.Column(0), tk.Sticky("ew"), tk.Padx(4), tk.Ipady(6))
	tk.Grid(u.confirm, tk.Row(3), tk.Column(1), tk.Sticky("ew"), tk.Padx(4), tk.Ipady(6))
	u.generate = button(frame.Window, "Generate password", func() {
		q.cancelAutoStart()
		const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%&*+-="
		var value strings.Builder
		var bytes [64]byte
		for value.Len() < 28 {
			if _, e := rand.Read(bytes[:]); e != nil {
				u.status.Configure(tk.Txt("Secure randomness is unavailable."))
				return
			}
			for _, b := range bytes {
				if int(b) < 256-256%len(alphabet) && value.Len() < 28 {
					value.WriteByte(alphabet[int(b)%len(alphabet)])
				}
			}
		}
		clear(bytes[:])
		u.password.Configure(tk.Textvariable(value.String()), tk.Show(""))
		u.confirm.Configure(tk.Textvariable(value.String()))
		u.showing = true
		if len(q.paths) > 0 && !q.decryptOnly {
			u.confirm.Configure(tk.Textvariable(""))
			u.status.Configure(tk.Txt("Copy and save this generated password, then paste it into Confirm password. Encryption will start automatically; the password cannot be recovered."))
			tk.Focus(u.password)
		} else {
			u.status.Configure(tk.Txt("Record this password securely before adding files. It cannot be recovered."))
		}
	})
	u.show = button(frame.Window, "Show / hide", func() {
		show := ""
		if u.showing {
			show = "•"
		}
		u.showing = !u.showing
		u.password.Configure(tk.Show(show))
		u.confirm.Configure(tk.Show(show))
	})
	tk.Grid(u.generate, tk.Row(4), tk.Column(0), tk.Sticky("w"), tk.Pady(8))
	tk.Grid(u.show, tk.Row(4), tk.Column(1), tk.Sticky("w"))
	simple := frame.Frame(tk.Background(paper))
	tk.Grid(simple, tk.Row(6), tk.Columnspan(2), tk.Sticky("nsew"))
	tk.GridColumnConfigure(simple, 0, tk.Weight(1))
	tk.GridRowConfigure(simple, 0, tk.Weight(1))
	q.drop = simple.Label(tk.Txt("Drop files or folders here"), tk.Font("Helvetica", 18, "bold"), tk.Background("white"), tk.Foreground(navy), tk.Highlightbackground(blue), tk.Highlightcolor(blue), tk.Highlightthickness(2), tk.Padx(20), tk.Pady(35), tk.Takefocus(true), tk.Wraplength(690))
	tk.Grid(q.drop, tk.Row(0), tk.Sticky("nsew"), tk.Pady(6))
	tk.Grid(label(simple.Window, "Originals are zipped and encrypted. Recognized packages are unlocked automatically."), tk.Row(1), tk.Sticky("w"), tk.Pady(5))
	choices := simple.Frame(tk.Background(paper))
	tk.Grid(choices, tk.Row(2), tk.Sticky("ew"), tk.Pady(6))
	q.files = button(choices.Window, "Choose files…", func() {
		q.cancelAutoStart()
		q.accept(tk.GetOpenFile(tk.Title("Choose files"), tk.Multiple(true)))
	})
	q.folder = button(choices.Window, "Choose folder…", func() {
		q.cancelAutoStart()
		path := tk.ChooseDirectory(tk.Title("Choose folder"))
		if path != "" {
			q.accept([]string{path})
		}
	})
	q.start = button(simple.Window, "Process selected", q.begin)
	q.start.Configure(tk.Background(blue), tk.Foreground("white"))
	tk.Pack(q.files, q.folder, tk.Side("left"), tk.Padx(3))
	tk.Grid(q.start, tk.Row(5), tk.Sticky("ew"), tk.Pady(5))
	tk.Bind(q.drop, "<Return>", tk.Command(func() { q.files.Invoke() }))
	for _, entry := range []*tk.EntryWidget{u.password, u.confirm} {
		tk.Bind(entry, "<Return>", tk.Command(q.begin))
		for _, event := range []string{"<KeyRelease>", "<<Paste>>", "<<Cut>>", "<ButtonRelease-2>"} {
			tk.Bind(entry, event, tk.Command(q.credentialsChanged))
		}
	}
	// Advanced options contain no action or data-type choice.
	advanced := frame.Frame(tk.Background(paper))
	tk.Grid(advanced, tk.Row(5), tk.Columnspan(2), tk.Sticky("ew"))
	u.action = advanced.TCombobox(tk.Textvariable("Encrypt"))
	u.source = advanced.TCombobox(tk.Textvariable("Text"))
	u.algorithm = advanced.TCombobox(tk.Values([]string{"AES-256-GCM", "XChaCha20-Poly1305"}), tk.Textvariable("AES-256-GCM"), tk.State("readonly"), tk.Width(24), font)
	tk.Pack(label(advanced.Window, "Note algorithm"), u.algorithm, tk.Side("left"), tk.Padx(5))
	// Keep helper controls for lifecycle locking; the main interface saves automatically.
	u.choose = button(advanced.Window, "Choose", u.chooseInput)
	u.selected = label(advanced.Window, "")
	u.run = button(advanced.Window, "Run", u.begin)
	u.save = button(advanced.Window, "Save", u.saveResult)
	u.save.Configure(tk.State("disabled"))
	tk.GridRemove(advanced.Window)
	tk.Grid(label(simple.Window, "Or type or paste text · encrypted text is detected automatically"), tk.Row(3), tk.Sticky("w"), tk.Pady(4))
	u.note = simple.Text(font, tk.Height(3), tk.Width(50), tk.Wrap("word"), tk.Background("white"), tk.Foreground(navy), tk.Padx(8), tk.Pady(6), tk.Undo(false))
	tk.Grid(u.note, tk.Row(4), tk.Sticky("ew"))
	tk.Bind(u.note, "<KeyRelease>", tk.Command(func() {
		if !u.busy && u.note.Text() != "" {
			q.cancelAutoStart()
			q.paths = nil
			q.drop.Configure(tk.Txt("Drop files or folders here"))
			u.status.Configure(tk.Txt("Text selected. CipherEnc will detect whether to encrypt or decrypt it."))
		}
	}))
	progress := frame.Frame(tk.Background(paper))
	tk.Grid(progress, tk.Row(7), tk.Columnspan(2), tk.Sticky("ew"), tk.Pady(8))
	tk.GridColumnConfigure(progress, 1, tk.Weight(1))
	tk.GridColumnConfigure(progress, 3, tk.Weight(1))
	tk.Grid(label(progress.Window, "Current item"), tk.Row(0), tk.Column(0), tk.Padx(5))
	q.current = progress.TProgressbar(tk.Mode("determinate"), tk.Maximum(100), tk.Value(0), tk.Length(150))
	tk.Grid(q.current, tk.Row(0), tk.Column(1), tk.Sticky("ew"))
	tk.Grid(label(progress.Window, "All items"), tk.Row(0), tk.Column(2), tk.Padx(8))
	q.all = progress.TProgressbar(tk.Mode("determinate"), tk.Maximum(100), tk.Value(0), tk.Length(150))
	tk.Grid(q.all, tk.Row(0), tk.Column(3), tk.Sticky("ew"))
	q.stop = button(progress.Window, "Stop", func() { q.cancelled.Store(true); u.status.Configure(tk.Txt("Stopping after the current safe step…")) })
	q.stop.Configure(tk.State("disabled"))
	tk.Grid(q.stop, tk.Row(0), tk.Column(4), tk.Padx(8))
	u.status = label(frame.Window, "Ready. Choose files or enter and confirm your password first.")
	u.status.Configure(tk.Background("#eaf2fc"), tk.Padx(10), tk.Pady(10), tk.Wraplength(760))
	tk.Grid(u.status, tk.Row(8), tk.Columnspan(2), tk.Sticky("ew"), tk.Pady(5))
	footer := frame.Frame(tk.Background(paper))
	tk.Grid(footer, tk.Row(9), tk.Columnspan(2), tk.Sticky("ew"))
	advancedVisible := false
	q.toggle = button(footer.Window, "Advanced options", func() {
		if u.busy {
			return
		}
		advancedVisible = !advancedVisible
		if advancedVisible {
			tk.Grid(advanced)
			q.toggle.Configure(tk.Txt("Hide advanced options"))
		} else {
			tk.GridRemove(advanced.Window)
			q.toggle.Configure(tk.Txt("Advanced options"))
		}
	})
	saveSlot := footer.Frame(tk.Background(paper))
	u.save = button(saveSlot.Window, "Save result…", u.saveResult)
	u.save.Configure(tk.State("disabled"))
	u.clear = button(footer.Window, "Clear", func() {
		if !u.busy {
			u.clearWork()
		}
	})
	tk.Pack(q.toggle, u.clear, tk.Side("left"), tk.Padx(3))
	tk.Pack(saveSlot, tk.Side("left"), tk.Padx(3))
	tk.Grid(label(frame.Window, "Originals stay in place. After verification, you can choose best-effort shredding."), tk.Row(10), tk.Columnspan(2), tk.Sticky("w"), tk.Pady(8))
	bridge := &tkdnd.Bridge{}
	defer bridge.Close()
	if _, e := tk.RegisterExtension("CipherEncDrop", bridge); e != nil {
		return e
	}
	if e := tk.InitializeExtension("CipherEncDrop"); e != nil {
		u.status.Configure(tk.Txt("Drag-and-drop unavailable: " + e.Error() + ". Use Choose files or Choose folder."))
	} else if e = bridge.Bind(q.drop, func(paths []string) { q.accept(paths) }); e != nil {
		return e
	}
	if !bridge.Loaded {
		u.status.Configure(tk.Txt("Drag-and-drop could not load. Use Choose files or Choose folder."))
	}
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
		_ = os.MkdirAll(smokeDir, 0700)
		path := filepath.Join(smokeDir, "Lesson notes.txt")
		_ = os.WriteFile(path, []byte("A private lesson stays on this device.\n\n"), 0600)
		go func() {
			time.Sleep(1200 * time.Millisecond)
			tk.PostEvent(func() {
				u.password.Configure(tk.Textvariable("Synthetic UI smoke passphrase!"))
				u.confirm.Configure(tk.Textvariable("Synthetic UI smoke passphrase!"))
				q.accept([]string{path})
			}, false)
			time.Sleep(4 * time.Second)
			tk.PostEvent(func() {
				report := map[string]any{"native_window": true, "browser_required": false, "ui_encryption": q.completed, "original_kept": false, "native_drop_loaded": bridge.Loaded}
				original, e := os.ReadFile(path)
				report["original_kept"] = e == nil && string(original) == "A private lesson stays on this device.\n\n"
				restored, e := decryptAdjacent(path+".cen", "Synthetic UI smoke passphrase!")
				if e == nil {
					b, e := os.ReadFile(filepath.Join(restored, "Lesson notes.txt"))
					report["trailing_newline_round_trip"] = e == nil && string(b) == string(original)
				}
				if report["ui_encryption"] != true || report["trailing_newline_round_trip"] != true || !bridge.Loaded {
					err = fmt.Errorf("Native UI archive round trip failed")
				}
				b, _ := json.MarshalIndent(report, "", "  ")
				_ = os.WriteFile(filepath.Join(smokeDir, "native-ui.json"), b, 0600)
			}, false)
			tk.PostEvent(func() {
				u.note.Insert("1.0", "Native text with trailing lines.\n\n")
				u.password.Configure(tk.Textvariable("Synthetic UI smoke passphrase!"))
				u.confirm.Configure(tk.Textvariable("Synthetic UI smoke passphrase!"))
				q.paths = nil
				q.begin()
			}, false)
			time.Sleep(3 * time.Second)
			tk.PostEvent(func() {
				reportPath := filepath.Join(smokeDir, "native-ui.json")
				data, _ := os.ReadFile(reportPath)
				report := map[string]any{}
				_ = json.Unmarshal(data, &report)
				packed, e := os.ReadFile(filepath.Join(smokeDir, "native-note.cen"))
				var restored []vaultFile
				if e == nil {
					restored, e = openPackage(packed, "Synthetic UI smoke passphrase!")
				}
				report["text_ui_encryption"] = e == nil && len(restored) == 1 && string(restored[0].Data) == "Native text with trailing lines.\n\n" && strings.HasPrefix(u.note.Text(), "PRISMVAULT3.")
				if report["text_ui_encryption"] != true {
					err = fmt.Errorf("Native text UI round trip failed: %v", e)
				}
				data, _ = json.MarshalIndent(report, "", "  ")
				_ = os.WriteFile(reportPath, data, 0600)
			}, false)
			time.Sleep(4 * time.Second)
			tk.PostEvent(func() { u.clearWork(); tk.Destroy(tk.App) }, false)
		}()
	}
	tk.App.Center().Wait()
	return err
}
