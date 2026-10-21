// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import "testing"

func TestDrawLine(t *testing.T) {
	c := MustCanvas(5, 5)
	c.DrawLine(0, 0, 4, 4, '*')
	for i := 0; i < 5; i++ {
		if c.Char(i, i) != '*' {
			t.Fatalf("diagonal missing at %d", i)
		}
	}
	c.DrawLine(4, 0, 0, 0, '-')
	for i := 0; i < 5; i++ {
		if c.Char(i, 0) != '-' {
			t.Fatalf("horizontal missing at %d", i)
		}
	}
}

func TestDrawBox(t *testing.T) {
	c := MustCanvas(6, 4)
	c.DrawBox(1, 1, 4, 2, '#')
	for _, p := range []Point{{1, 1}, {4, 1}, {1, 2}, {4, 2}, {2, 1}, {3, 2}} {
		if c.Char(p.X, p.Y) != '#' {
			t.Fatalf("box missing at %+v", p)
		}
	}
}

func TestFillBox(t *testing.T) {
	c := MustCanvas(5, 5)
	c.FillBox(1, 1, 3, 2, 'x')
	for y := 1; y <= 2; y++ {
		for x := 1; x <= 3; x++ {
			if c.Char(x, y) != 'x' {
				t.Fatalf("fill missing at %d,%d", x, y)
			}
		}
	}
	if c.Char(0, 0) != ' ' || c.Char(4, 4) != ' ' {
		t.Fatal("fill leaked outside the box")
	}
}

func TestDrawCircle(t *testing.T) {
	c := MustCanvas(11, 11)
	c.DrawCircle(5, 5, 4, '*')
	for _, p := range []Point{{5, 1}, {5, 9}, {1, 5}, {9, 5}} {
		if c.Char(p.X, p.Y) != '*' {
			t.Fatalf("circle missing at %+v", p)
		}
	}
	if c.Char(5, 5) != ' ' {
		t.Fatal("circle outline filled the centre")
	}
}

func TestFillTriangle(t *testing.T) {
	c := MustCanvas(7, 7)
	c.FillTriangle(1, 1, 5, 1, 3, 5, '#')
	if c.Char(3, 2) != '#' || c.Char(3, 4) != '#' {
		t.Fatal("triangle interior not filled")
	}
	if c.Char(0, 0) != ' ' {
		t.Fatal("triangle leaked outside its bounds")
	}
}
