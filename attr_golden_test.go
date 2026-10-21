// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import "testing"

// Golden values produced by libcaca's caca_set_color_argb() and
// caca_set_color_ansi(), verified against a C build of the reference.
func TestAttrGolden(t *testing.T) {
	argb := []struct {
		fg, bg uint16
		want   uint32
	}{
		{0xf00f, 0xf0f0, 0xe1e38070},
		{0xf0f0, 0xf00f, 0xe01f8780},
		{0x0fff, 0x0000, 0x02007ff0},
		{0x0000, 0x0fff, 0x1ffc0800},
		{0x1234, 0x5678, 0x4cf011a0},
		{0x8888, 0x8888, 0x91124440},
		{0x8fff, 0x8fff, 0x9ffe7ff0},
		{0xffff, 0xffff, 0xfffffff0},
	}
	for _, tc := range argb {
		got := uint32(ARGB16Attr(tc.fg, tc.bg, 0))
		if got != tc.want {
			t.Errorf("ARGB16Attr(%04x,%04x) = %#08x, want %#08x", tc.fg, tc.bg, got, tc.want)
		}
	}

	ansi := []struct {
		fg, bg Color
		want   uint32
	}{
		{Black, Black, 0x01000400},
		{Red, Black, 0x01000440},
		{Red, Blue, 0x01040440},
		{Default, Transparent, 0x01800500},
		{0x20, Black, 0x01000600},
	}
	for _, tc := range ansi {
		if got := uint32(ANSIAttr(tc.fg, tc.bg, 0)); got != tc.want {
			t.Errorf("ANSIAttr(%v,%v) = %#08x, want %#08x", tc.fg, tc.bg, got, tc.want)
		}
	}

	c := MustCanvas(3, 1)
	_ = c.SetColor(Red, Blue)
	c.PutChar(0, 0, 'a')
	_ = c.SetColor(Black, White)
	c.PutChar(1, 0, 'b')
	_ = c.SetColor(Default, Transparent)
	c.PutChar(2, 0, 'c')
	got := []uint32{uint32(c.Attr(0, 0)), uint32(c.Attr(1, 0)), uint32(c.Attr(2, 0))}
	if got[0] != 0x01040440 || got[1] != 0x013c0400 || got[2] != 0x01800500 {
		t.Fatalf("cell attrs = %#08x", got)
	}
	c.Invert()
	got = []uint32{uint32(c.Attr(0, 0)), uint32(c.Attr(1, 0)), uint32(c.Attr(2, 0))}
	if got[0] != 0x010b044f || got[1] != 0x0133040f || got[2] != 0x018f050f {
		t.Fatalf("inverted attrs = %#08x", got)
	}
}

// TestNativeGolden compares the native encoder against a .caca file produced
// by a C build of libcaca for the same canvas.
func TestNativeGolden(t *testing.T) {
	c := MustCanvas(3, 2)
	_ = c.SetColor(Red, Blue)
	c.PutString(0, 0, "ab")
	_ = c.SetColor(Green, Black)
	c.PutString(0, 1, "世界")

	want := []byte{
		0xca, 0xca, 0x43, 0x56, 0x00, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x30, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x04, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x61, 0x01, 0x04, 0x04, 0x40, 0x00, 0x00, 0x00, 0x62,
		0x01, 0x04, 0x04, 0x40, 0x00, 0x00, 0x00, 0x20, 0x01, 0x80, 0x05, 0x00, 0x00, 0x00, 0x4e, 0x16,
		0x01, 0x00, 0x04, 0x20, 0x00, 0x0f, 0xff, 0xfe, 0x01, 0x00, 0x04, 0x20, 0x00, 0x00, 0x00, 0x20,
		0x01, 0x00, 0x04, 0x20,
	}
	got, err := c.Encode("caca")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("native length %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("native byte %d = %#02x, want %#02x", i, got[i], want[i])
		}
	}
}
