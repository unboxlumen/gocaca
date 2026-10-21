// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"image"
	"image/color"
	"testing"
)

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			var c color.RGBA
			switch {
			case x < 4 && y < 4:
				c = color.RGBA{255, 0, 0, 255}
			case x >= 4 && y < 4:
				c = color.RGBA{0, 255, 0, 255}
			case x < 4:
				c = color.RGBA{0, 0, 255, 255}
			default:
				c = color.RGBA{255, 255, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func TestDitherDraw(t *testing.T) {
	d, err := NewDitherFromImage(testImage())
	if err != nil {
		t.Fatal(err)
	}
	if d.Width() != 8 || d.Height() != 8 {
		t.Fatalf("size = %dx%d", d.Width(), d.Height())
	}
	c := MustCanvas(8, 8)
	if err := d.Draw(c, 0, 0, 8, 8); err != nil {
		t.Fatal(err)
	}
	// The top-left quadrant is red: red channel should dominate.
	a := c.Attr(1, 1)
	fg, _, _ := a.RGB()
	if fg.R < fg.G || fg.R < fg.B {
		t.Fatalf("top-left is not reddish: %#v", fg)
	}
	// The bottom-right quadrant is white.
	a = c.Attr(6, 6)
	fg, _, _ = a.RGB()
	if fg.R < 0x80 || fg.G < 0x80 || fg.B < 0x80 {
		t.Fatalf("bottom-right is not bright: %#v", fg)
	}
}

func TestDitherOptions(t *testing.T) {
	d, err := NewDitherFromImage(testImage())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ascii", "shades", "blocks"} {
		if err := d.SetCharset(name); err != nil {
			t.Fatalf("charset %q: %v", name, err)
		}
	}
	if err := d.SetCharset("bogus"); err == nil {
		t.Fatal("bogus charset accepted")
	}
	for _, name := range []string{"mono", "gray", "8", "16", "fullgray", "full8", "full16"} {
		if err := d.SetColor(name); err != nil {
			t.Fatalf("colour %q: %v", name, err)
		}
	}
	for _, name := range []string{"none", "ordered2", "ordered4", "ordered8", "random", "fstein"} {
		if err := d.SetAlgorithm(name); err != nil {
			t.Fatalf("algorithm %q: %v", name, err)
		}
	}
	if err := d.SetGamma(0); err == nil {
		t.Fatal("zero gamma accepted")
	}
	if err := d.SetBrightness(-1); err == nil {
		t.Fatal("negative brightness accepted")
	}
}

func TestDitherNoPanic(t *testing.T) {
	d, err := NewDitherFromImage(testImage())
	if err != nil {
		t.Fatal(err)
	}
	c := MustCanvas(1, 1)
	for _, algo := range []string{"none", "ordered2", "ordered4", "ordered8", "random", "fstein"} {
		for _, cs := range []string{"ascii", "shades", "blocks"} {
			for _, col := range []string{"mono", "gray", "8", "16", "fullgray", "full8", "full16"} {
				if err := d.SetAlgorithm(algo); err != nil {
					t.Fatal(err)
				}
				if err := d.SetCharset(cs); err != nil {
					t.Fatal(err)
				}
				if err := d.SetColor(col); err != nil {
					t.Fatal(err)
				}
				if err := d.Draw(c, 0, 0, 1, 1); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestDitherInvalid(t *testing.T) {
	if _, err := NewDither(nil, 0, 1, 24); err == nil {
		t.Fatal("zero width accepted")
	}
	if _, err := NewDither(nil, 1, 1, 16); err == nil {
		t.Fatal("16 bpp accepted")
	}
	if _, err := NewDither(make([]byte, 2), 1, 1, 24); err == nil {
		t.Fatal("short buffer accepted")
	}
}
