package main

// Reading files straight out of the game's own archives (Packed_DX12\*.bin), so the program
// doesn't have to carry any game data itself.

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// the game's archives, in load order: a file in a later archive replaces the same file in an earlier one
var gameArchives = []string{"DLC1", "Initial", "Remainder", "Patch"}

type binFile struct {
	off  uint64
	size uint32
}

type binChunk struct {
	uoff  uint64 // offset in the uncompressed stream
	usize uint32
	coff  uint64 // offset of the compressed chunk in the .bin
	csize uint32
}

type gameArchive struct {
	name   string
	f      *os.File
	files  map[uint64]binFile
	chunks []binChunk
}

type archiveSet struct {
	archives []*gameArchive
	where    map[uint64]*gameArchive // path hash -> archive that wins
}

func openArchive(dir, name string) (*gameArchive, error) {
	f, err := os.Open(filepath.Join(dir, name+".bin"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s.bin isn't in the game folder - this tool needs Horizon Zero Dawn Complete Edition (original PC version, not Remastered)", name)
		}
		return nil, fmt.Errorf("couldn't open %s.bin: %v", name, err)
	}
	hdr := make([]byte, 40)
	if _, err := f.ReadAt(hdr, 0); err != nil {
		f.Close()
		return nil, fmt.Errorf("%s.bin: %v", name, err)
	}
	magic := binary.LittleEndian.Uint32(hdr)
	if magic != 0x20304050 {
		f.Close()
		if magic == 0x21304050 {
			return nil, fmt.Errorf("%s.bin is encrypted - this version of the game isn't supported", name)
		}
		return nil, fmt.Errorf("%s.bin doesn't look like a Horizon Zero Dawn archive", name)
	}
	nFiles := binary.LittleEndian.Uint64(hdr[24:])
	nChunks := binary.LittleEndian.Uint32(hdr[32:])
	if nFiles > 1<<22 || nChunks > 1<<24 {
		f.Close()
		return nil, fmt.Errorf("%s.bin: unexpected index size", name)
	}
	idx := make([]byte, 32*(int(nFiles)+int(nChunks)))
	if _, err := f.ReadAt(idx, 40); err != nil {
		f.Close()
		return nil, fmt.Errorf("%s.bin index: %v", name, err)
	}
	a := &gameArchive{name: name, f: f, files: make(map[uint64]binFile, nFiles)}
	for i := 0; i < int(nFiles); i++ {
		e := idx[32*i:]
		a.files[binary.LittleEndian.Uint64(e[8:])] = binFile{binary.LittleEndian.Uint64(e[16:]), binary.LittleEndian.Uint32(e[24:])}
	}
	base := 32 * int(nFiles)
	a.chunks = make([]binChunk, nChunks)
	for i := range a.chunks {
		e := idx[base+32*i:]
		a.chunks[i] = binChunk{binary.LittleEndian.Uint64(e), binary.LittleEndian.Uint32(e[8:]),
			binary.LittleEndian.Uint64(e[16:]), binary.LittleEndian.Uint32(e[24:])}
	}
	return a, nil
}

func openArchiveSet(packedDir string) (*archiveSet, error) {
	s := &archiveSet{where: map[uint64]*gameArchive{}}
	for _, n := range gameArchives {
		a, err := openArchive(packedDir, n)
		if err != nil {
			s.close()
			return nil, err
		}
		s.archives = append(s.archives, a)
		for h := range a.files {
			s.where[h] = a // later archives win
		}
	}
	return s, nil
}

func (s *archiveSet) close() {
	for _, a := range s.archives {
		a.f.Close()
	}
}

// read returns one whole file from the archives
func (s *archiveSet) read(path string) ([]byte, error) {
	h := archiveHash(path)
	a, ok := s.where[h]
	if !ok {
		return nil, fmt.Errorf("not in the game archives: %s", path)
	}
	fe := a.files[h]
	start, end := fe.off, fe.off+uint64(fe.size)
	i := sort.Search(len(a.chunks), func(i int) bool { return a.chunks[i].uoff+uint64(a.chunks[i].usize) > start })
	out := make([]byte, 0, fe.size)
	for ; i < len(a.chunks) && a.chunks[i].uoff < end; i++ {
		c := a.chunks[i]
		comp := make([]byte, c.csize)
		if _, err := a.f.ReadAt(comp, int64(c.coff)); err != nil {
			return nil, fmt.Errorf("%s.bin: %v", a.name, err)
		}
		raw, err := decompressChunk(comp, int(c.usize))
		if err != nil {
			return nil, fmt.Errorf("%s (%s.bin): %v", path, a.name, err)
		}
		lo, hi := uint64(0), uint64(c.usize)
		if start > c.uoff {
			lo = start - c.uoff
		}
		if end < c.uoff+uint64(c.usize) {
			hi = end - c.uoff
		}
		out = append(out, raw[lo:hi]...)
	}
	if len(out) != int(fe.size) {
		return nil, fmt.Errorf("%s: read %d of %d bytes", path, len(out), fe.size)
	}
	return out, nil
}

// archiveHash: the archive id of a game path given with its extension ("x/y.core", "x/y.core.stream")
func archiveHash(p string) uint64 { return pathHash(p) }
