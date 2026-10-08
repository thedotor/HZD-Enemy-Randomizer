package main

import (
	"encoding/binary"
	"math/bits"
)

func fmix64(k uint64) uint64 {
	k ^= k >> 33
	k *= 0xff51afd7ed558ccd
	k ^= k >> 33
	k *= 0xc4ceb9fe1a85ec53
	k ^= k >> 33
	return k
}

// MurmurHash3 x64 128, returns first 64 bits
func murmur3(data []byte, seed uint64) uint64 {
	const c1, c2 = 0x87c37b91114253d5, 0x4cf5ad432745937f
	h1, h2 := seed, seed
	n := len(data)
	nb := n / 16
	for i := 0; i < nb; i++ {
		k1 := binary.LittleEndian.Uint64(data[i*16:])
		k2 := binary.LittleEndian.Uint64(data[i*16+8:])
		k1 *= c1
		k1 = bits.RotateLeft64(k1, 31)
		k1 *= c2
		h1 ^= k1
		h1 = bits.RotateLeft64(h1, 27)
		h1 += h2
		h1 = h1*5 + 0x52dce729
		k2 *= c2
		k2 = bits.RotateLeft64(k2, 33)
		k2 *= c1
		h2 ^= k2
		h2 = bits.RotateLeft64(h2, 31)
		h2 += h1
		h2 = h2*5 + 0x38495ab5
	}
	tail := make([]byte, 16)
	copy(tail, data[nb*16:])
	r := n & 15
	if r > 8 {
		k2 := binary.LittleEndian.Uint64(tail[8:]) & ((uint64(1) << (8 * uint(r-8))) - 1)
		k2 *= c2
		k2 = bits.RotateLeft64(k2, 33)
		k2 *= c1
		h2 ^= k2
	}
	if r > 0 {
		m := 8
		if r < 8 {
			m = r
		}
		k1 := binary.LittleEndian.Uint64(tail)
		if m < 8 {
			k1 &= (uint64(1) << (8 * uint(m))) - 1
		}
		k1 *= c1
		k1 = bits.RotateLeft64(k1, 31)
		k1 *= c2
		h1 ^= k1
	}
	h1 ^= uint64(n)
	h2 ^= uint64(n)
	h1 += h2
	h2 += h1
	h1 = fmix64(h1)
	h2 = fmix64(h2)
	h1 += h2
	return h1
}

func pathHash(p string) uint64 {
	return murmur3(append([]byte(p), 0), 42)
}
