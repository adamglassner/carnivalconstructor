//go:build js

package main

// The browser build is only used for automated screenshots.

import "syscall/js"

func fatal(text string) { panic(text) }

func singleInstance() bool { return true }

func testQuery() string { return js.Global().Get("location").Get("search").String() }
