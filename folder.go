package main

import (
	"path/filepath"
	"strings"
)

// resolveGameFolder accepts the Packed_DX12 folder, the game's install folder, or a path with quotes.
func resolveGameFolder(p string) (string, bool) {
	p = strings.Trim(strings.TrimSpace(p), `"'`)
	if p == "" {
		return "", false
	}
	if isPacked(p) {
		return p, true
	}
	if isPacked(filepath.Join(p, "Packed_DX12")) {
		return filepath.Join(p, "Packed_DX12"), true
	}
	if strings.EqualFold(filepath.Base(p), "Initial.bin") && isPacked(filepath.Dir(p)) {
		return filepath.Dir(p), true
	}
	return p, false
}
