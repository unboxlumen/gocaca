// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"image/color"
	"testing"
)

func TestANSIAttr(t *testing.T) {
	a := ANSIAttr(Red, Blue, Bold|Underline)
	if a.FG() != Red || a.BG() != Blue {
		t.Fatalf("colours = %v/%v, want Red/Blue", a.FG(), a.BG())
	}
	if a.Style() != Bold|Underline {
		t.Fatalf("style = %v", a.Style())
	}
	if a.IsRGB() {
		t.Fatal("ANSI attribute reported as RGB")
	}
	if a.WithStyle(Italics).Style() != Italics {
		t.Fatal("WithStyle did not replace the mask")
	}
}

func TestRGBAttr(t *testing.T) {
	fg := color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}
	bg := color.RGBA{R: 0xab, G: 0xcd, B: 0xef, A: 0xff}
	a := RGBAttr(fg, bg, Bold)
	if !a.IsRGB() {
		t.Fatal("RGB attribute not marked")
	}
	gotFG, gotBG, ok := a.RGB()
	if !ok {
		t.Fatal("RGB() returned ok=false")
	}
	// libcaca drops the low bit of each 4-bit component: 0x12 -> 0x11,
	// 0x34 -> 0x33, 0x56 -> 0x44 (14-bit repacking).
	if gotFG.R != 0x11 || gotFG.G != 0x33 || gotFG.B != 0x44 {
		t.Fatalf("quantised fg = %#v", gotFG)
	}
	if gotBG.R != 0xaa || gotBG.G != 0xcc || gotBG.B != 0xee {
		t.Fatalf("quantised bg = %#v", gotBG)
	}
	// libcaca reports the nearest ANSI colour for truecolour components;
	// it does not collapse them to Default.
	if a.FG() >= 0x10 || a.BG() >= 0x10 {
		t.Fatalf("nearest ANSI = %v/%v", a.FG(), a.BG())
	}
}

func TestSetColorValidation(t *testing.T) {
	c := MustCanvas(1, 1)
	if err := c.SetColor(Color(0x30), Black); err == nil {
		t.Fatal("out-of-range colour accepted")
	}
	if err := c.SetColor(Transparent, Black); err != nil {
		t.Fatalf("Transparent rejected: %v", err)
	}
	if err := c.SetColor(Default, Transparent); err != nil {
		t.Fatalf("Default/Transparent rejected: %v", err)
	}
}

func TestSetAttrStyleOnly(t *testing.T) {
	c := MustCanvas(1, 1)
	if err := c.SetColor(Red, Blue); err != nil {
		t.Fatal(err)
	}
	c.SetAttr(Attr(Bold))
	if c.CurrentAttr().FG() != Red || c.CurrentAttr().BG() != Blue {
		t.Fatal("style-only SetAttr clobbered colours")
	}
	if c.CurrentAttr().Style() != Bold {
		t.Fatal("style was not applied")
	}
}
