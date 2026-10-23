//go:build setup

package main

import _ "embed"

// Written by build.ps1 before the setup build.
var (
	//go:embed payload/Rabbit.exe.gz
	rabbitGz []byte
	//go:embed payload/uninstall.exe.gz
	uninstallGz []byte
)

const hasPayload = true
