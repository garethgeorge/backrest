//go:build !windows

package main

import "flag"

func main() {
	flag.Parse()
	runApp()
}
