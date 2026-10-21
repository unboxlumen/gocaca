// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"bytes"
	"testing"
)

func TestANSITruecolor(t *testing.T) {
	c := MustCanvas(0, 0)
	if err := c.Decode([]byte("\x1b[38;2;255;0;0mX"), "utf8"); err != nil {
		t.Fatal(err)
	}
	a := c.Attr(0, 0)
	if !a.IsRGB() {
		t.Fatal("truecolour attr not marked as RGB")
	}
	fg, _, _ := a.RGB()
	if fg.R < 0xe0 || fg.G > 0x0f || fg.B > 0x0f {
		t.Fatalf("truecolour fg = %#v", fg)
	}
}

func TestANSI256(t *testing.T) {
	c := MustCanvas(0, 0)
	if err := c.Decode([]byte("\x1b[38;5;196mX"), "utf8"); err != nil {
		t.Fatal(err)
	}
	fg, _, _ := c.Attr(0, 0).RGB()
	if fg.R < 0xe0 || fg.G > 0x0f || fg.B > 0x0f {
		t.Fatalf("256-colour fg = %#v", fg)
	}
	c2 := MustCanvas(0, 0)
	if err := c2.Decode([]byte("\x1b[48;5;21mX"), "utf8"); err != nil {
		t.Fatal(err)
	}
	_, bg, _ := c2.Attr(0, 0).RGB()
	if bg.B < 0xe0 || bg.R > 0x0f || bg.G > 0x0f {
		t.Fatalf("256-colour bg = %#v", bg)
	}
}

func TestANSICP437(t *testing.T) {
	c := MustCanvas(0, 0)
	if err := c.Decode([]byte{0xb0}, "ansi"); err != nil {
		t.Fatal(err)
	}
	if got := c.Char(0, 0); got != 0x2591 {
		t.Fatalf("CP437 decode = %#x", got)
	}
	out, err := c.Encode("ansi")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte{0xb0}) {
		t.Fatalf("ANSI encode = %q", out)
	}
}

func TestTextTabs(t *testing.T) {
	c := MustCanvas(0, 0)
	if err := c.Decode([]byte("a\tb"), "text"); err != nil {
		t.Fatal(err)
	}
	if c.Width() != 9 || c.Char(8, 0) != 'b' {
		t.Fatalf("tab decode size=%d char=%q", c.Width(), c.Char(8, 0))
	}
}

func TestManage(t *testing.T) {
	c := MustCanvas(2, 2)
	if err := c.Manage(nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Manage(nil); err != ErrCanvasManaged {
		t.Fatalf("double Manage = %v", err)
	}
	if err := c.SetBoundaries(0, 0, 1, 1); err != ErrCanvasManaged {
		t.Fatalf("SetBoundaries on managed canvas = %v", err)
	}
	if err := c.Resize(3, 3); err != nil {
		t.Fatalf("Resize without callback = %v", err)
	}
	if err := c.Unmanage(); err != nil {
		t.Fatal(err)
	}

	c2 := MustCanvas(2, 2)
	if err := c2.Manage(func() bool { return false }); err != nil {
		t.Fatal(err)
	}
	if err := c2.Resize(3, 3); err != ErrCanvasManaged {
		t.Fatalf("vetoed Resize = %v", err)
	}
}
