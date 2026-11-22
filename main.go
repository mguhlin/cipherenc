// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

const version = "1.1.0"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	smoke := flag.String("self-test", "", "run synthetic crypto/filesystem checks in this folder and exit")
	uiSmoke := flag.String("ui-smoke", "", "run native-window smoke check in this folder and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("CipherEnc", version, "native desktop")
		return
	}
	if *smoke != "" {
		if err := selfTest(*smoke); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	runtime.LockOSThread()
	if err := runGUI(*uiSmoke); err != nil {
		showMessage(err.Error())
		os.Exit(1)
	}
}
