//go:build !windows && !js

package main

import (
	"fmt"
	"os"
)

func fatal(text string) {
	fmt.Fprintln(os.Stderr, text)
	os.Exit(1)
}

func singleInstance() bool { return true }

func testQuery() string { return os.Getenv("CC_TEST") }
