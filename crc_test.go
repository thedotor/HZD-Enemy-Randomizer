package main

import "testing"

func TestCRC(t *testing.T) {
	if v := decimaCRC("entities/spawnsetups/robots/antelope/antelope"); v != 0x36f4e816 {
		t.Fatalf("crc %x", v)
	}
}
func TestCRC2(t *testing.T) {
	if v := decimaCRC("entities/spawnsetups/robots/greywolf/greywolf"); v != 0x68cb4446 {
		t.Fatalf("crc %x", v)
	}
}
