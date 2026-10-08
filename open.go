package main

import (
	"os/exec"
	"runtime"
)

func openFile(p string) {
	if runtime.GOOS == "windows" {
		exec.Command("notepad.exe", p).Start()
	}
}
