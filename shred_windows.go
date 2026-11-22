// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"io/fs"
	"os"
)

func singleLink(f *os.File, _ fs.FileInfo) error {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &info); err != nil {
		return err
	}
	if info.NumberOfLinks != 1 {
		return fmt.Errorf("Shredding is unavailable for hard-linked files. Keep the original.")
	}
	return nil
}
