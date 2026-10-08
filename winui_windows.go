//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

// Native Windows calls only: no PowerShell, no helper processes.
var (
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	pShellEx = shell32.NewProc("ShellExecuteW")
	pBrowse  = shell32.NewProc("SHBrowseForFolderW")
	pGetPath = shell32.NewProc("SHGetPathFromIDListW")
	pCoInit  = ole32.NewProc("CoInitializeEx")
	pCoUn    = ole32.NewProc("CoUninitialize")
	pCoFree  = ole32.NewProc("CoTaskMemFree")
	pFg      = user32.NewProc("GetForegroundWindow")
	pSend    = user32.NewProc("SendMessageW")
)

func openBrowser(url string) {
	verb, _ := syscall.UTF16PtrFromString("open")
	u, _ := syscall.UTF16PtrFromString(url)
	pShellEx.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(u)), 0, 0, 1)
}

type browseInfo struct {
	owner       uintptr
	root        uintptr
	displayName *uint16
	title       *uint16
	flags       uint32
	callback    uintptr
	lParam      uintptr
	image       int32
}

var browseStart *uint16

var browseCB = syscall.NewCallback(func(hwnd, msg, lp, data uintptr) uintptr {
	const bffmInitialized, bffmSetSelectionW = 1, 0x400 + 103
	if msg == bffmInitialized && browseStart != nil {
		pSend.Call(hwnd, bffmSetSelectionW, 1, uintptr(unsafe.Pointer(browseStart)))
	}
	return 0
})

// browseFolder shows the standard Windows "Browse for folder" window ("" if cancelled).
func browseFolder(start string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pCoInit.Call(0, 2) // apartment threaded
	defer pCoUn.Call()
	name := make([]uint16, 260)
	title, _ := syscall.UTF16PtrFromString("Choose the Horizon Zero Dawn folder (or its Packed_DX12 folder)")
	browseStart = nil
	if start != "" {
		browseStart, _ = syscall.UTF16PtrFromString(start)
	}
	owner, _, _ := pFg.Call()
	bi := browseInfo{owner: owner, displayName: &name[0], title: title,
		flags: 0x1 | 0x40 | 0x200, callback: browseCB} // folders only, new style, no "new folder" button
	pidl, _, _ := pBrowse.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", nil
	}
	defer pCoFree.Call(pidl)
	path := make([]uint16, 1024)
	if ok, _, _ := pGetPath.Call(pidl, uintptr(unsafe.Pointer(&path[0]))); ok == 0 {
		return "", nil
	}
	return syscall.UTF16ToString(path), nil
}
