//go:build !windows && !oozdev

package main

import "fmt"

// Only Windows has the game (and its Oodle library); other platforms can build but not read game files.
func initDecompressor(gameDir string) error {
	return fmt.Errorf("reading the game's archives needs Windows")
}

func decompressChunk(comp []byte, rawLen int) ([]byte, error) {
	return nil, fmt.Errorf("no decompressor on this platform")
}
