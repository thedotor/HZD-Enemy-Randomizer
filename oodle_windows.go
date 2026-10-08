//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"syscall"
	"unsafe"
)

// The game ships Oodle (oo2core_*_win64.dll) next to HorizonZeroDawn.exe. We borrow it to read
// the game's archives - the program itself contains no decompressor and no game files.
var oodleProc *syscall.Proc

func initDecompressor(gameDir string) error {
	if oodleProc != nil {
		return nil
	}
	var found []string
	for _, d := range []string{gameDir, filepath.Dir(gameDir)} {
		m, _ := filepath.Glob(filepath.Join(d, "oo2core_*_win64.dll"))
		found = append(found, m...)
	}
	if len(found) == 0 {
		return fmt.Errorf("couldn't find the game's oo2core_*_win64.dll next to HorizonZeroDawn.exe")
	}
	sort.Strings(found)
	dll, err := syscall.LoadDLL(found[len(found)-1])
	if err != nil {
		return fmt.Errorf("couldn't load %s: %v", filepath.Base(found[len(found)-1]), err)
	}
	p, err := dll.FindProc("OodleLZ_Decompress")
	if err != nil {
		return fmt.Errorf("%s has no OodleLZ_Decompress", filepath.Base(found[len(found)-1]))
	}
	oodleProc = p
	return nil
}

func decompressChunk(comp []byte, rawLen int) ([]byte, error) {
	if oodleProc == nil {
		return nil, fmt.Errorf("decompressor not loaded")
	}
	out := make([]byte, rawLen+64)
	// OodleLZ_Decompress(comp, compLen, raw, rawLen, fuzzSafe=1, checkCRC=0, verbosity=0,
	//                    decBufBase, decBufSize, fpCallback, userData, decoderMem, decoderMemSize, threadPhase=3)
	r, _, _ := oodleProc.Call(uintptr(unsafe.Pointer(&comp[0])), uintptr(len(comp)),
		uintptr(unsafe.Pointer(&out[0])), uintptr(rawLen), 1, 0, 0, 0, 0, 0, 0, 0, 0, 3)
	if int(r) != rawLen {
		return nil, fmt.Errorf("decompression failed (%d of %d bytes)", int(r), rawLen)
	}
	return out[:rawLen], nil
}
