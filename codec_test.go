// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"bytes"
	"strings"
	"testing"
)

func TestTextRoundTrip(t *testing.T) {
	c := MustCanvas(4, 2)
	c.PutString(0, 0, "abc")
	c.PutString(0, 1, "de")
	data, err := c.Encode("text")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "abc \nde  \n" {
		t.Fatalf("text = %q", data)
	}
	d := MustCanvas(0, 0)
	if err := d.Decode(data, "text"); err != nil {
		t.Fatal(err)
	}
	if d.Width() != 4 || d.Height() != 2 {
		t.Fatalf("decoded size = %dx%d", d.Width(), d.Height())
	}
	if d.Char(0, 0) != 'a' || d.Char(1, 1) != 'e' {
		t.Fatal("text round trip lost content")
	}
}

func TestANSIRoundTrip(t *testing.T) {
	c := MustCanvas(3, 1)
	if err := c.SetColor(Red, Blue); err != nil {
		t.Fatal(err)
	}
	c.SetAttr(c.CurrentAttr().WithStyle(Bold))
	c.PutChar(0, 0, 'a')
	if err := c.SetColor(Green, Black); err != nil {
		t.Fatal(err)
	}
	c.PutChar(1, 0, 'b')
	data, err := c.Encode("ansi")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("\x1b[")) {
		t.Fatal("ANSI output has no escape sequence")
	}
	d := MustCanvas(0, 0)
	if err := d.Decode(data, "ansi"); err != nil {
		t.Fatal(err)
	}
	if d.Char(0, 0) != 'a' || d.Char(1, 0) != 'b' {
		t.Fatal("ANSI round trip lost characters")
	}
	a := d.Attr(0, 0)
	if a.FG() != Red || a.BG() != Blue || a.Style() != Bold {
		t.Fatalf("ANSI attribute = fg %v bg %v style %v", a.FG(), a.BG(), a.Style())
	}
}

func TestUTF8Codec(t *testing.T) {
	c := MustCanvas(4, 1)
	c.PutString(0, 0, "世界")
	data, err := c.Encode("utf8")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("世界")) {
		t.Fatalf("utf8 output = %q", data)
	}
	d := MustCanvas(0, 0)
	if err := d.Decode(data, "utf8"); err != nil {
		t.Fatal(err)
	}
	if d.Char(0, 0) != '世' || d.Char(2, 0) != '界' {
		t.Fatal("utf8 round trip lost fullwidth cells")
	}
}

func TestNativeRoundTrip(t *testing.T) {
	c := MustCanvas(2, 2)
	c.PutChar(0, 0, 'a')
	c.NewFrame(1)
	c.SetFrame(1)
	c.PutChar(1, 1, 'b')
	data, err := c.Encode("caca")
	if err != nil {
		t.Fatal(err)
	}
	d := MustCanvas(0, 0)
	if err := d.Decode(data, "caca"); err != nil {
		t.Fatal(err)
	}
	if d.FrameCount() != 2 {
		t.Fatalf("frames = %d", d.FrameCount())
	}
	if d.Char(0, 0) != 'a' {
		t.Fatal("frame 0 content lost")
	}
	if err := d.SetFrame(1); err != nil {
		t.Fatal(err)
	}
	if d.Char(1, 1) != 'b' {
		t.Fatal("frame 1 content lost")
	}
}

func TestCodecAutodetect(t *testing.T) {
	c := MustCanvas(2, 1)
	c.PutChar(0, 0, 'a')
	native, _ := c.Encode("caca")
	for _, tc := range []struct {
		data []byte
		want rune
	}{
		{native, 'a'},
		{[]byte("hi"), 'h'},
		{[]byte("\x1b[31mR\x1b[0m"), 'R'},
	} {
		d := MustCanvas(0, 0)
		if err := d.Decode(tc.data, ""); err != nil {
			t.Fatalf("autodetect %q: %v", tc.data, err)
		}
		if d.Char(0, 0) != tc.want {
			t.Fatalf("autodetect %q = %q, want %q", tc.data, d.Char(0, 0), tc.want)
		}
	}
}

func TestBadNativeLeavesCanvas(t *testing.T) {
	c := MustCanvas(2, 2)
	c.PutChar(0, 0, 'x')
	if err := c.Decode([]byte("not a native file"), "caca"); err == nil {
		t.Fatal("malformed native input accepted")
	}
	if c.Char(0, 0) != 'x' {
		t.Fatal("failed decode modified the canvas")
	}
}

func TestFormats(t *testing.T) {
	got := strings.Join(Formats(), ",")
	if got != "text,ansi,utf8,caca" {
		t.Fatalf("Formats = %q", got)
	}
}
