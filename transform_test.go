// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import "testing"

func canvasString(c *Canvas) string {
	b := make([]rune, 0, c.Width()*c.Height())
	for y := 0; y < c.Height(); y++ {
		for x := 0; x < c.Width(); x++ {
			if ch := c.Char(x, y); ch != Fullwidth {
				b = append(b, ch)
			}
		}
	}
	return string(b)
}

func TestFlipFlopInvolutions(t *testing.T) {
	c := MustCanvas(3, 2)
	c.PutString(0, 0, "abc")
	c.PutString(0, 1, "def")
	orig := canvasString(c)
	c.Flip()
	c.Flip()
	if got := canvasString(c); got != orig {
		t.Fatalf("flip twice = %q, want %q", got, orig)
	}
	c.Flop()
	c.Flop()
	if got := canvasString(c); got != orig {
		t.Fatalf("flop twice = %q, want %q", got, orig)
	}
}

func TestRotateFourTimes(t *testing.T) {
	c := MustCanvas(3, 2)
	c.PutString(0, 0, "abc")
	c.PutString(0, 1, "def")
	orig := canvasString(c)
	c.RotateLeft()
	if c.Width() != 2 || c.Height() != 3 {
		t.Fatalf("rotate size = %dx%d", c.Width(), c.Height())
	}
	c.RotateLeft()
	c.RotateLeft()
	c.RotateLeft()
	if c.Width() != 3 || c.Height() != 2 {
		t.Fatalf("four rotations size = %dx%d", c.Width(), c.Height())
	}
	if got := canvasString(c); got != orig {
		t.Fatalf("four rotations = %q, want %q", got, orig)
	}
}

func TestRotateDirection(t *testing.T) {
	c := MustCanvas(2, 2)
	c.PutString(0, 0, "ab")
	c.PutString(0, 1, "cd")
	c.RotateRight()
	// Clockwise: a b / c d  ->  c a / d b
	if got := canvasString(c); got != "cadb" {
		t.Fatalf("rotate right = %q, want %q", got, "cadb")
	}
	c2 := MustCanvas(2, 2)
	c2.PutString(0, 0, "ab")
	c2.PutString(0, 1, "cd")
	c2.RotateLeft()
	// Counter-clockwise: a b / c d -> b d / a c
	if got := canvasString(c2); got != "bdac" {
		t.Fatalf("rotate left = %q, want %q", got, "bdac")
	}
}

func TestStretch(t *testing.T) {
	c := MustCanvas(2, 2)
	c.PutString(0, 0, "ab")
	c.PutString(0, 1, "cd")
	c.StretchHorizontal()
	if c.Width() != 4 || c.Height() != 2 {
		t.Fatalf("stretch horizontal size = %dx%d", c.Width(), c.Height())
	}
	if got := canvasString(c); got != "aabbccdd" {
		t.Fatalf("stretch horizontal = %q", got)
	}
	c.StretchVertical()
	if c.Width() != 4 || c.Height() != 4 {
		t.Fatalf("stretch vertical size = %dx%d", c.Width(), c.Height())
	}
	if got := canvasString(c); got != "aabbaabbccddccdd" {
		t.Fatalf("stretch vertical = %q", got)
	}
}

func TestInvert(t *testing.T) {
	// libcaca's invert is attr ^= 0x000f000f: it complements the low four
	// bits of each colour component, it does not swap fg and bg.
	c := MustCanvas(1, 1)
	if err := c.SetColor(Red, Blue); err != nil {
		t.Fatal(err)
	}
	c.PutChar(0, 0, 'x')
	before := c.Attr(0, 0)
	c.Invert()
	after := c.Attr(0, 0)
	if after != before^0x000f000f {
		t.Fatalf("invert = %#x, want %#x", after, before^0x000f000f)
	}
	c.Invert()
	if got := c.Attr(0, 0); got != before {
		t.Fatalf("double invert = %#x, want %#x", got, before)
	}
}
