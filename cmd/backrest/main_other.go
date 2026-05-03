//go:build !windows && !linux && !darwin

package main

import "flag"

func main() {
	flag.Parse()
	runApp()
}
