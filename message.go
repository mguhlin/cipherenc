//go:build !windows

package main

import "fmt"

func showMessage(message string) { fmt.Println("CipherEnc:", message) }
