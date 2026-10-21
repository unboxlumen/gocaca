// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import "testing"

func TestNewCanvas(t *testing.T) {
	c, err := NewCanvas(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if c.Width() != 0 || c.Height() != 0 {
		t.Fatalf("size = %dx%d", c.Width(), c.Height())
	}
	if _, err := NewCanvas(-1, 0); err == nil {
		t.Fatal("negative width accepted")
	}
	if _, err := NewCanvas(maxInt/2+1, 2); err == nil {
		t.Fatal("overflowing size accepted")
	}
}

func TestCanvasPutGet(t *testing.T) {
	c := MustCanvas(3, 2)
	if !c.PutChar(1, 1, 'x') {
		t.Fatal("PutChar failed")
	}
	if got := c.Char(1, 1); got != 'x' {
		t.Fatalf("Char = %q", got)
	}
	if c.PutChar(-1, 0, 'x') {
		t.Fatal("out-of-bounds PutChar succeeded")
	}
	if got := c.Char(-1, 0); got != ' ' {
		t.Fatalf("out-of-bounds Char = %q", got)
	}
	c.PutChar(0, 0, 0)
	if got := c.Char(0, 0); got != ' ' {
		t.Fatalf("NUL = %q", got)
	}
}

func TestResizePreserves(t *testing.T) {
	c := MustCanvas(2, 2)
	c.PutChar(0, 0, 'a')
	c.PutChar(1, 0, 'b')
	c.PutChar(0, 1, 'c')
	if err := c.Resize(4, 3); err != nil {
		t.Fatal(err)
	}
	if c.Width() != 4 || c.Height() != 3 {
		t.Fatalf("size = %dx%d", c.Width(), c.Height())
	}
	for _, tc := range []struct {
		x, y int
		want rune
	}{{0, 0, 'a'}, {1, 0, 'b'}, {0, 1, 'c'}, {3, 2, ' '}} {
		if got := c.Char(tc.x, tc.y); got != tc.want {
			t.Fatalf("Char(%d,%d) = %q, want %q", tc.x, tc.y, got, tc.want)
		}
	}
	if err := c.Resize(1, 1); err != nil {
		t.Fatal(err)
	}
	if got := c.Char(0, 0); got != 'a' {
		t.Fatalf("shrunk canvas lost content: %q", got)
	}
	if err := c.Resize(-1, 1); err == nil {
		t.Fatal("negative resize accepted")
	}
}

func TestFullwidthCells(t *testing.T) {
	c := MustCanvas(4, 1)
	if !c.PutChar(1, 0, '世') {
		t.Fatal("fullwidth put failed")
	}
	if got := c.Char(1, 0); got != '世' {
		t.Fatalf("lead = %q", got)
	}
	if got := c.Char(2, 0); got != Fullwidth {
		t.Fatalf("marker = %#x", got)
	}
	c.PutChar(2, 0, 'x')
	if got := c.Char(1, 0); got != ' ' {
		t.Fatalf("overwriting marker left lead %q", got)
	}
	if got := c.Char(2, 0); got != 'x' {
		t.Fatalf("overwriting marker = %q", got)
	}
	c2 := MustCanvas(3, 1)
	c2.PutChar(0, 0, '世')
	c2.PutChar(0, 0, 'y')
	if got := c2.Char(1, 0); got != ' ' {
		t.Fatalf("overwriting lead left marker %#x", got)
	}
}

func TestFullwidthAtEdge(t *testing.T) {
	c := MustCanvas(2, 1)
	if !c.PutChar(1, 0, '世') {
		t.Fatal("PutChar at edge failed")
	}
	if got := c.Char(1, 0); got != ' ' {
		t.Fatalf("edge fullwidth = %q, want space", got)
	}
}

func TestPutString(t *testing.T) {
	c := MustCanvas(10, 3)
	n := c.PutString(0, 0, "ab\ncd")
	if n != 4 {
		t.Fatalf("PutString returned %d, want 4", n)
	}
	if c.Char(0, 0) != 'a' || c.Char(1, 0) != 'b' ||
		c.Char(0, 1) != 'c' || c.Char(1, 1) != 'd' {
		t.Fatal("newline handling failed")
	}
	c.PutString(1, 2, "\tx")
	if c.Char(8, 2) != 'x' {
		t.Fatalf("tab did not advance to column 8: %q", c.Char(8, 2))
	}
}

func TestFrames(t *testing.T) {
	c := MustCanvas(2, 2)
	c.PutChar(0, 0, 'a')
	if c.NewFrame(1) != 1 {
		t.Fatal("NewFrame returned wrong index")
	}
	if c.FrameCount() != 2 {
		t.Fatalf("FrameCount = %d", c.FrameCount())
	}
	if c.CurrentFrame() != 0 {
		t.Fatalf("NewFrame changed active frame to %d", c.CurrentFrame())
	}
	if err := c.SetFrame(1); err != nil {
		t.Fatal(err)
	}
	if c.Char(0, 0) != 'a' {
		t.Fatal("new frame was not a copy")
	}
	c.PutChar(0, 0, 'b')
	if err := c.SetFrame(0); err != nil {
		t.Fatal(err)
	}
	if c.Char(0, 0) != 'a' {
		t.Fatal("frames share storage")
	}
	if err := c.SetFrameName(0, "first"); err != nil {
		t.Fatal(err)
	}
	if c.FrameName(0) != "first" {
		t.Fatalf("FrameName = %q", c.FrameName(0))
	}
	if err := c.FreeFrame(1); err != nil {
		t.Fatal(err)
	}
	if c.FrameCount() != 1 {
		t.Fatal("FreeFrame failed")
	}
	if err := c.FreeFrame(0); err == nil {
		t.Fatal("removed the last frame")
	}
}

func TestDirtyRects(t *testing.T) {
	c := MustCanvas(8, 8)
	c.ClearDirtyRects()
	c.PutChar(1, 1, 'x')
	if c.DirtyRectCount() == 0 {
		t.Fatal("PutChar did not mark dirty")
	}
	if _, _, w, h, ok := c.DirtyRect(0); !ok || w != 1 || h != 1 {
		t.Fatalf("dirty rect = %dx%d ok=%v", w, h, ok)
	}
	c.DisableDirtyRects()
	c.ClearDirtyRects()
	c.PutChar(2, 2, 'y')
	if c.DirtyRectCount() != 0 {
		t.Fatal("dirty tracking was not disabled")
	}
	if err := c.EnableDirtyRects(); err != nil {
		t.Fatal(err)
	}
	c.PutChar(3, 3, 'z')
	if c.DirtyRectCount() == 0 {
		t.Fatal("dirty tracking did not re-enable")
	}
	c.ClearDirtyRects()
	if c.DirtyRectCount() != 0 {
		t.Fatal("ClearDirtyRects failed")
	}
}

func TestBlit(t *testing.T) {
	dst := MustCanvas(4, 4)
	src := MustCanvas(2, 2)
	src.PutChar(0, 0, 'a')
	src.PutChar(1, 1, 'b')
	if err := dst.Blit(1, 1, src, nil); err != nil {
		t.Fatal(err)
	}
	if dst.Char(1, 1) != 'a' || dst.Char(2, 2) != 'b' {
		t.Fatal("blit did not copy")
	}
	mask := MustCanvas(2, 2)
	mask.PutChar(0, 0, 'x')
	dst2 := MustCanvas(4, 4)
	if err := dst2.Blit(0, 0, src, mask); err != nil {
		t.Fatal(err)
	}
	if dst2.Char(0, 0) != 'a' {
		t.Fatal("mask allowed cell was not copied")
	}
	if dst2.Char(1, 1) != ' ' {
		t.Fatal("mask blocked cell was copied")
	}
	if err := dst2.Blit(0, 0, src, MustCanvas(3, 3)); err != ErrMaskSize {
		t.Fatalf("mask size error = %v", err)
	}
}

func TestSetBoundaries(t *testing.T) {
	c := MustCanvas(4, 4)
	c.PutChar(1, 1, 'a')
	c.PutChar(2, 2, 'b')
	if err := c.SetBoundaries(1, 1, 2, 2); err != nil {
		t.Fatal(err)
	}
	if c.Width() != 2 || c.Height() != 2 {
		t.Fatalf("size = %dx%d", c.Width(), c.Height())
	}
	if c.Char(0, 0) != 'a' || c.Char(1, 1) != 'b' {
		t.Fatal("boundaries did not shift content")
	}
}
