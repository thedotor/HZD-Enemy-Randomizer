package main

// Decoding the few texture formats the map tiles and map-marker icons use.

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

const (
	pfRGBA8888 = 12
	pfBC1      = 66
	pfBC3      = 68
)

func rgb565(c uint16) (uint8, uint8, uint8) {
	r, g, b := (c>>11)&31, (c>>5)&63, c&31
	return uint8(r * 255 / 31), uint8(g * 255 / 63), uint8(b * 255 / 31)
}

// colour block shared by BC1 and BC3 (8 bytes); bc1Alpha allows the 1-bit transparent colour
func decodeColourBlock(blk []byte, bc1Alpha bool) [16]color.NRGBA {
	c0, c1 := binary.LittleEndian.Uint16(blk), binary.LittleEndian.Uint16(blk[2:])
	var pal [4]color.NRGBA
	r0, g0, b0 := rgb565(c0)
	r1, g1, b1 := rgb565(c1)
	pal[0] = color.NRGBA{r0, g0, b0, 255}
	pal[1] = color.NRGBA{r1, g1, b1, 255}
	if c0 > c1 || !bc1Alpha {
		pal[2] = color.NRGBA{uint8((2*int(r0) + int(r1)) / 3), uint8((2*int(g0) + int(g1)) / 3), uint8((2*int(b0) + int(b1)) / 3), 255}
		pal[3] = color.NRGBA{uint8((int(r0) + 2*int(r1)) / 3), uint8((int(g0) + 2*int(g1)) / 3), uint8((int(b0) + 2*int(b1)) / 3), 255}
	} else {
		pal[2] = color.NRGBA{uint8((int(r0) + int(r1)) / 2), uint8((int(g0) + int(g1)) / 2), uint8((int(b0) + int(b1)) / 2), 255}
		pal[3] = color.NRGBA{0, 0, 0, 0}
	}
	bits := binary.LittleEndian.Uint32(blk[4:])
	var out [16]color.NRGBA
	for i := 0; i < 16; i++ {
		out[i] = pal[(bits>>(2*i))&3]
	}
	return out
}

func decodeBC3Alpha(blk []byte) [16]uint8 {
	a0, a1 := int(blk[0]), int(blk[1])
	var pal [8]int
	pal[0], pal[1] = a0, a1
	if a0 > a1 {
		for i := 1; i <= 6; i++ {
			pal[i+1] = ((7-i)*a0 + i*a1) / 7
		}
	} else {
		for i := 1; i <= 4; i++ {
			pal[i+1] = ((5-i)*a0 + i*a1) / 5
		}
		pal[6], pal[7] = 0, 255
	}
	var bits uint64
	for i := 0; i < 6; i++ {
		bits |= uint64(blk[2+i]) << (8 * i)
	}
	var out [16]uint8
	for i := 0; i < 16; i++ {
		out[i] = uint8(pal[(bits>>(3*i))&7])
	}
	return out
}

func decodeTexture(w, h, format int, data []byte) (*image.NRGBA, error) {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	switch format {
	case pfRGBA8888:
		if len(data) < w*h*4 {
			return nil, fmt.Errorf("texture data too short")
		}
		copy(img.Pix, data[:w*h*4])
		return img, nil
	case pfBC1, pfBC3:
		bs := 8
		if format == pfBC3 {
			bs = 16
		}
		bw, bh := (w+3)/4, (h+3)/4
		if len(data) < bw*bh*bs {
			return nil, fmt.Errorf("texture data too short")
		}
		for by := 0; by < bh; by++ {
			for bx := 0; bx < bw; bx++ {
				blk := data[(by*bw+bx)*bs:]
				var px [16]color.NRGBA
				if format == pfBC1 {
					px = decodeColourBlock(blk, true)
				} else {
					px = decodeColourBlock(blk[8:], false)
					al := decodeBC3Alpha(blk)
					for i := range px {
						px[i].A = al[i]
					}
				}
				for i := 0; i < 16; i++ {
					x, y := bx*4+i%4, by*4+i/4
					if x < w && y < h {
						img.SetNRGBA(x, y, px[i])
					}
				}
			}
		}
		return img, nil
	}
	return nil, fmt.Errorf("texture format %d not supported", format)
}

// texture header (32 bytes) + texture data header (16 bytes) as stored in Texture / UITexture objects
type texInfo struct {
	w, h, mips, format int
	internal           []byte
	extSize            int
	extMips            int
}

func readTexture(b []byte, o int) (texInfo, int, error) {
	var t texInfo
	if o+48 > len(b) {
		return t, o, fmt.Errorf("texture header cut short")
	}
	t.w = int(binary.LittleEndian.Uint16(b[o+2:]) & 0x3fff)
	t.h = int(binary.LittleEndian.Uint16(b[o+4:]) & 0x3fff)
	t.mips = int(b[o+8])
	t.format = int(b[o+9])
	o += 32
	intSize := int(binary.LittleEndian.Uint32(b[o+4:]))
	t.extSize = int(binary.LittleEndian.Uint32(b[o+8:]))
	t.extMips = int(binary.LittleEndian.Uint32(b[o+12:]))
	o += 16
	if t.extSize > 0 { // data source: location string (no checksum), offset, length
		if o+4 > len(b) {
			return t, o, fmt.Errorf("texture source cut short")
		}
		n := int(binary.LittleEndian.Uint32(b[o:]))
		o += 4 + n + 16
	}
	if o+intSize > len(b) {
		intSize = len(b) - o
	}
	if intSize < 0 {
		return t, o, fmt.Errorf("texture data cut short")
	}
	t.internal = b[o : o+intSize]
	return t, o + intSize, nil
}

// skipString: Decima string = u32 length, then (if not empty) u32 checksum + bytes
func skipString(b []byte, o int) int {
	n := int(binary.LittleEndian.Uint32(b[o:]))
	if n == 0 {
		return o + 4
	}
	return o + 8 + n
}
