// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build !windows

package main

import (
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

func singleLink(_ *os.File, info fs.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Nlink != 1 {
		return fmt.Errorf("Shredding is unavailable for hard-linked files. Keep the original.")
	}
	return nil
}
