// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import "testing"

func TestUTF8RoundTrip(t *testing.T) {
	in := "hello, 世界 🎨"
	rs := UTF8ToUTF32(in)
	if got := UTF32ToUTF8(rs); got != in {
		t.Fatalf("round trip = %q, want %q", got, in)
	}
}

func TestUTF8Invalid(t *testing.T) {
	rs := UTF8ToUTF32("\xff")
	if len(rs) != 1 || rs[0] != 0xfffd {
		t.Fatalf("invalid UTF-8 = %#v", rs)
	}
}

func TestCP437(t *testing.T) {
	cases := []struct {
		b byte
		r rune
	}{
		{0x41, 'A'},
		{0x7f, 0x7f},
		{0xb0, 0x2591},
		{0xc4, 0x2500},
		{0xff, 0x00a0},
	}
	for _, tc := range cases {
		if got := CP437ToUTF32(tc.b); got != tc.r {
			t.Fatalf("CP437ToUTF32(%#x) = %#x, want %#x", tc.b, got, tc.r)
		}
		if got := UTF32ToCP437(tc.r); got != tc.b {
			t.Fatalf("UTF32ToCP437(%#x) = %#x, want %#x", tc.r, got, tc.b)
		}
	}
	if got := UTF32ToCP437('世'); got != '?' {
		t.Fatalf("unrepresentable rune = %q", got)
	}
}

func TestASCII(t *testing.T) {
	if UTF32ToASCII('A') != 'A' {
		t.Fatal("ASCII identity failed")
	}
	if UTF32ToASCII(0x2500) != '-' {
		t.Fatal("box drawing did not map to '-'")
	}
	if UTF32ToASCII(0x1f600) != '?' {
		t.Fatal("unknown rune did not map to '?'")
	}
}

func TestFullwidth(t *testing.T) {
	for _, r := range []rune{'世', '界', 'あ', '한'} {
		if !IsFullwidth(r) {
			t.Fatalf("%U should be fullwidth", r)
		}
	}
	for _, r := range []rune{'A', 'é', '─'} {
		if IsFullwidth(r) {
			t.Fatalf("%U should not be fullwidth", r)
		}
	}
}
