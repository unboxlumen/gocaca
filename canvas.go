// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"errors"
	"fmt"
	"image/color"
	"unicode/utf8"
)

var (
	ErrInvalidSize   = errors.New("libcaca: invalid canvas size")
	ErrSizeOverflow  = errors.New("libcaca: canvas size overflow")
	ErrInvalidColor  = errors.New("libcaca: invalid colour")
	ErrInvalidFrame  = errors.New("libcaca: invalid frame")
	ErrLastFrame     = errors.New("libcaca: cannot remove the last frame")
	ErrMaskSize      = errors.New("libcaca: mask size does not match source")
	ErrCanvasManaged = errors.New("libcaca: canvas is managed")
)

// Rect is a cell rectangle.
type Rect struct {
	X, Y, Width, Height int
}

// Point is a cell coordinate.
type Point struct{ X, Y int }

type frame struct {
	width, height int
	chars         []rune
	attrs         []Attr
	x, y          int
	handlex       int
	handley       int
	curattr       Attr
	name          string
}

// Canvas is a colour ASCII-art canvas. It is not safe for concurrent use.
type Canvas struct {
	frames      []*frame
	frame       int
	cur         *frame
	dirty       []Rect
	dirtyOff    int
	autoinc     int
	managed     bool
	resizeCheck func() bool
}

const maxInt = int(^uint(0) >> 1)

func checkedArea(width, height int) (int, error) {
	if width < 0 || height < 0 {
		return 0, ErrInvalidSize
	}
	if width != 0 && height > maxInt/width {
		return 0, ErrSizeOverflow
	}
	return width * height, nil
}

func newFrame(width, height int, attr Attr, name string) *frame {
	f := &frame{
		width:   width,
		height:  height,
		chars:   make([]rune, width*height),
		attrs:   make([]Attr, width*height),
		curattr: attr,
		name:    name,
	}
	for i := range f.chars {
		f.chars[i] = ' '
		f.attrs[i] = attr
	}
	return f
}

// NewCanvas creates a canvas. Zero width or height is allowed.
func NewCanvas(width, height int) (*Canvas, error) {
	if _, err := checkedArea(width, height); err != nil {
		return nil, err
	}
	// libcaca initialises new canvases to CACA_DEFAULT on CACA_TRANSPARENT.
	def := ANSIAttr(Default, Transparent, 0)
	f := newFrame(width, height, def, "frame#00000000")
	return &Canvas{frames: []*frame{f}, cur: f}, nil
}

// MustCanvas is NewCanvas, panicking on error.
func MustCanvas(width, height int) *Canvas {
	c, err := NewCanvas(width, height)
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Canvas) Width() int  { return c.cur.width }
func (c *Canvas) Height() int { return c.cur.height }

// Resize changes every frame to the requested size. The top-left content is
// preserved and new cells use the frame's current attribute.
func (c *Canvas) Resize(width, height int) error {
	if _, err := checkedArea(width, height); err != nil {
		return err
	}
	if c.resizeCheck != nil && !c.resizeCheck() {
		return ErrCanvasManaged
	}
	for _, f := range c.frames {
		ch, at := make([]rune, width*height), make([]Attr, width*height)
		for i := range ch {
			ch[i] = ' '
			at[i] = f.curattr
		}
		copyW := min(width, f.width)
		copyH := min(height, f.height)
		for y := 0; y < copyH; y++ {
			copy(ch[y*width:y*width+copyW], f.chars[y*f.width:y*f.width+copyW])
			copy(at[y*width:y*width+copyW], f.attrs[y*f.width:y*f.width+copyW])
		}
		f.width, f.height, f.chars, f.attrs = width, height, ch, at
		if f.x > width {
			f.x = width
		}
		if f.y > height {
			f.y = height
		}
	}
	c.cur = c.frames[c.frame]
	c.clipDirty()
	if !c.dirtyDisabled() {
		c.addDirtyRect(0, 0, width, height)
	}
	return nil
}

// Manage locks resizing. The callback, when non-nil, can veto a resize.
func (c *Canvas) Manage(callback func() bool) error {
	if c.managed {
		return ErrCanvasManaged
	}
	c.managed = true
	c.resizeCheck = callback
	return nil
}

// Unmanage releases a previous Manage call.
func (c *Canvas) Unmanage() error {
	if !c.managed {
		return ErrCanvasManaged
	}
	c.managed = false
	c.resizeCheck = nil
	return nil
}

// Clear fills the current frame with spaces using the current attribute.
func (c *Canvas) Clear() {
	for i := range c.cur.chars {
		c.cur.chars[i] = ' '
		c.cur.attrs[i] = c.cur.curattr
	}
	c.dirty = c.dirty[:0]
	if !c.dirtyDisabled() {
		c.addDirtyRect(0, 0, c.cur.width, c.cur.height)
	}
}

func (c *Canvas) putChar(x, y int, ch rune) (int, bool) {
	if ch == 0 {
		ch = ' '
	}
	if ch == Fullwidth {
		return 1, false
	}
	fw := IsFullwidth(ch)
	ret := 1
	if fw {
		ret = 2
	}
	if x >= c.cur.width || y < 0 || y >= c.cur.height {
		return ret, false
	}
	if x == -1 && fw {
		x, ch, fw = 0, ' ', false
	} else if x < 0 {
		return ret, false
	}

	idx := x + y*c.cur.width
	attr := c.cur.curattr
	xmin, xmax := x, x

	if x > 0 && c.cur.chars[idx] == Fullwidth {
		c.cur.chars[idx-1] = ' '
		xmin--
	}
	if fw {
		if x+1 == c.cur.width {
			ch = ' '
		} else {
			xmax++
			if x+2 < c.cur.width && c.cur.chars[idx+2] == Fullwidth {
				c.cur.chars[idx+2] = ' '
				xmax++
			}
			c.cur.chars[idx+1] = Fullwidth
			c.cur.attrs[idx+1] = attr
		}
	} else if x+1 != c.cur.width && c.cur.chars[idx+1] == Fullwidth {
		c.cur.chars[idx+1] = ' '
		xmax++
	}

	if !c.dirtyDisabled() && (c.cur.chars[idx] != ch || c.cur.attrs[idx] != attr) {
		c.addDirtyRect(xmin, y, xmax-xmin+1, 1)
	}
	c.cur.chars[idx] = ch
	c.cur.attrs[idx] = attr
	return ret, true
}

// PutChar writes a rune at (x,y). It reports whether the cell was inside
// the canvas. A fullwidth rune occupies two cells.
func (c *Canvas) PutChar(x, y int, ch rune) bool {
	_, ok := c.putChar(x, y, ch)
	return ok
}

// Char returns the rune at (x,y), or ' ' outside the canvas. The trailing
// half of a fullwidth rune returns Fullwidth.
func (c *Canvas) Char(x, y int) rune {
	if x < 0 || x >= c.cur.width || y < 0 || y >= c.cur.height {
		return ' '
	}
	return c.cur.chars[x+y*c.cur.width]
}

// PutAttr sets the attribute at (x,y) without changing the character. For a
// fullwidth pair, both cells receive the attribute.
func (c *Canvas) PutAttr(x, y int, a Attr) bool {
	if x < 0 || x >= c.cur.width || y < 0 || y >= c.cur.height {
		return false
	}
	idx := x + y*c.cur.width
	xmin, xmax := x, x
	if a < 0x10 {
		a = (c.cur.attrs[idx] &^ attrStyleMask) | a
	}
	c.cur.attrs[idx] = a
	if x > 0 && c.cur.chars[idx] == Fullwidth {
		c.cur.attrs[idx-1] = a
		xmin--
	} else if x+1 < c.cur.width && c.cur.chars[idx+1] == Fullwidth {
		c.cur.attrs[idx+1] = a
		xmax++
	}
	if !c.dirtyDisabled() {
		c.addDirtyRect(xmin, y, xmax-xmin+1, 1)
	}
	return true
}

// Attr returns the attribute at (x,y), or the current attribute outside the
// canvas.
func (c *Canvas) Attr(x, y int) Attr {
	if x < 0 || x >= c.cur.width || y < 0 || y >= c.cur.height {
		return c.cur.curattr
	}
	return c.cur.attrs[x+y*c.cur.width]
}

// SetAttr sets the current drawing attribute. Values below 0x10 are treated
// as style bits and preserve the current colours.
func (c *Canvas) SetAttr(a Attr) {
	if a < 0x10 {
		a = (c.cur.curattr &^ attrStyleMask) | a
	}
	c.cur.curattr = a
}

// CurrentAttr returns the current drawing attribute.
func (c *Canvas) CurrentAttr() Attr { return c.cur.curattr }

// SetColor sets the current ANSI colour pair, preserving style bits.
func (c *Canvas) SetColor(fg, bg Color) error {
	if !validColor(fg) || !validColor(bg) {
		return ErrInvalidColor
	}
	c.cur.curattr = (c.cur.curattr & attrStyleMask) | ANSIAttr(fg, bg, 0)
	return nil
}

// SetRGB sets the current truecolour pair, preserving style bits.
func (c *Canvas) SetRGB(fg, bg color.RGBA) {
	c.cur.curattr = (c.cur.curattr & attrStyleMask) | RGBAttr(fg, bg, 0)
}

// PutString writes a UTF-8 string. Newlines move to the next row, carriage
// returns reset the column, and tabs advance to the next multiple of eight.
// It returns the number of cells occupied by glyphs.
func (c *Canvas) PutString(x, y int, s string) int {
	written := 0
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if size == 0 {
			break
		}
		s = s[size:]
		switch r {
		case '\n':
			x, y = 0, y+1
			continue
		case '\r':
			x = 0
			continue
		case '\t':
			x = (x + 8) &^ 7
			continue
		}
		w := 1
		if IsFullwidth(r) {
			w = 2
		}
		if y >= 0 && y < c.cur.height && x >= -1 && x < c.cur.width {
			c.putChar(x, y, r)
		}
		x += w
		written += w
	}
	return written
}

// Printf formats and writes a string.
func (c *Canvas) Printf(x, y int, format string, args ...any) int {
	return c.PutString(x, y, fmt.Sprintf(format, args...))
}

// SetHandle sets the blit/cursor handle for the current frame.
func (c *Canvas) SetHandle(x, y int) {
	c.cur.handlex, c.cur.handley = x, y
}

// Handle returns the current handle.
func (c *Canvas) Handle() (int, int) { return c.cur.handlex, c.cur.handley }

// Blit copies src onto c at (x,y). A non-nil mask skips cells whose mask
// character is a space.
func (c *Canvas) Blit(x, y int, src *Canvas, mask *Canvas) error {
	if src == nil {
		return errors.New("libcaca: nil source canvas")
	}
	if mask != nil && (src.cur.width != mask.cur.width || src.cur.height != mask.cur.height) {
		return ErrMaskSize
	}
	x -= src.cur.handlex
	y -= src.cur.handley
	starti := max(0, -x)
	startj := max(0, -y)
	endi := min(src.cur.width, c.cur.width-x)
	endj := min(src.cur.height, c.cur.height-y)
	if starti >= endi || startj >= endj {
		return nil
	}
	for j := startj; j < endj; j++ {
		dstRow := (j + y) * c.cur.width
		srcRow := j * src.cur.width
		if starti+x > 0 && c.cur.chars[dstRow+starti+x] == Fullwidth {
			c.cur.chars[dstRow+starti+x-1] = ' '
		}
		if endi+x < c.cur.width && c.cur.chars[dstRow+endi+x] == Fullwidth {
			c.cur.chars[dstRow+endi+x] = ' '
		}
		for i := starti; i < endi; i++ {
			di := dstRow + i + x
			si := srcRow + i
			if mask != nil && mask.cur.chars[si] == ' ' {
				continue
			}
			if c.cur.chars[di] != src.cur.chars[si] || c.cur.attrs[di] != src.cur.attrs[si] {
				c.cur.chars[di] = src.cur.chars[si]
				c.cur.attrs[di] = src.cur.attrs[si]
				if !c.dirtyDisabled() {
					c.addDirtyRect(i+x, j+y, 1, 1)
				}
			}
		}
		if src.cur.chars[srcRow+starti] == Fullwidth {
			c.cur.chars[dstRow+starti+x] = ' '
		}
		if endi < src.cur.width && src.cur.chars[srcRow+endi] == Fullwidth && endi+x-1 >= 0 {
			c.cur.chars[dstRow+endi+x-1] = ' '
		}
	}
	return nil
}

// SetBoundaries crops or expands the canvas. Every frame is affected.
func (c *Canvas) SetBoundaries(x, y, width, height int) error {
	if _, err := checkedArea(width, height); err != nil {
		return err
	}
	if c.managed {
		return ErrCanvasManaged
	}
	for _, f := range c.frames {
		nf := newFrame(width, height, f.curattr, f.name)
		nf.x, nf.y = f.x-x, f.y-y
		nf.handlex, nf.handley = f.handlex-x, f.handley-y
		nf.curattr = f.curattr
		for ny := 0; ny < height; ny++ {
			sy := ny + y
			if sy < 0 || sy >= f.height {
				continue
			}
			for nx := 0; nx < width; nx++ {
				sx := nx + x
				if sx < 0 || sx >= f.width {
					continue
				}
				nf.chars[ny*width+nx] = f.chars[sy*f.width+sx]
				nf.attrs[ny*width+nx] = f.attrs[sy*f.width+sx]
			}
		}
		*f = *nf
	}
	c.cur = c.frames[c.frame]
	c.dirty = c.dirty[:0]
	if !c.dirtyDisabled() {
		c.addDirtyRect(0, 0, width, height)
	}
	return nil
}

func (c *Canvas) FrameCount() int   { return len(c.frames) }
func (c *Canvas) CurrentFrame() int { return c.frame }
func (c *Canvas) FrameName(i int) string {
	if i < 0 || i >= len(c.frames) {
		return ""
	}
	return c.frames[i].name
}

// SetFrame activates frame i.
func (c *Canvas) SetFrame(i int) error {
	if i < 0 || i >= len(c.frames) {
		return ErrInvalidFrame
	}
	if i == c.frame {
		return nil
	}
	c.frame = i
	c.cur = c.frames[i]
	if !c.dirtyDisabled() {
		c.addDirtyRect(0, 0, c.cur.width, c.cur.height)
	}
	return nil
}

// NewFrame inserts a copy of the current frame at i and returns the new
// frame's index. The active frame stays active.
func (c *Canvas) NewFrame(i int) int {
	if i < 0 {
		i = 0
	}
	if i > len(c.frames) {
		i = len(c.frames)
	}
	old := c.cur
	nf := &frame{
		width:   old.width,
		height:  old.height,
		chars:   append([]rune(nil), old.chars...),
		attrs:   append([]Attr(nil), old.attrs...),
		x:       old.x,
		y:       old.y,
		handlex: old.handlex,
		handley: old.handley,
		curattr: old.curattr,
		name:    fmt.Sprintf("frame#%08x", c.autoinc+1),
	}
	c.autoinc++
	c.frames = append(c.frames, nil)
	copy(c.frames[i+1:], c.frames[i:])
	c.frames[i] = nf
	if c.frame >= i {
		c.frame++
	}
	c.cur = c.frames[c.frame]
	return i
}

// FreeFrame removes frame i. The last frame cannot be removed.
func (c *Canvas) FreeFrame(i int) error {
	if i < 0 || i >= len(c.frames) {
		return ErrInvalidFrame
	}
	if len(c.frames) == 1 {
		return ErrLastFrame
	}
	c.frames = append(c.frames[:i], c.frames[i+1:]...)
	if c.frame > i {
		c.frame--
	} else if c.frame == i {
		c.frame = 0
	}
	c.cur = c.frames[c.frame]
	if !c.dirtyDisabled() {
		c.addDirtyRect(0, 0, c.cur.width, c.cur.height)
	}
	return nil
}

// SetFrameName changes the name of frame i.
func (c *Canvas) SetFrameName(i int, name string) error {
	if i < 0 || i >= len(c.frames) {
		return ErrInvalidFrame
	}
	c.frames[i].name = name
	return nil
}

func (c *Canvas) dirtyDisabled() bool { return c.dirtyOff > 0 }

func (c *Canvas) DisableDirtyRects() { c.dirtyOff++ }

func (c *Canvas) EnableDirtyRects() error {
	if c.dirtyOff <= 0 {
		return errors.New("libcaca: dirty rectangles are not disabled")
	}
	c.dirtyOff--
	return nil
}

func (c *Canvas) DirtyRectCount() int { return len(c.dirty) }

func (c *Canvas) DirtyRect(i int) (x, y, width, height int, ok bool) {
	if i < 0 || i >= len(c.dirty) {
		return 0, 0, 0, 0, false
	}
	r := c.dirty[i]
	return r.X, r.Y, r.Width, r.Height, true
}

func (c *Canvas) ClearDirtyRects() { c.dirty = c.dirty[:0] }

func (c *Canvas) AddDirtyRect(x, y, width, height int) {
	if c.dirtyDisabled() {
		return
	}
	c.addDirtyRect(x, y, width, height)
}

func (c *Canvas) addDirtyRect(x, y, width, height int) {
	if x < 0 {
		width += x
		x = 0
	}
	if x+width > c.cur.width {
		width = c.cur.width - x
	}
	if y < 0 {
		height += y
		y = 0
	}
	if y+height > c.cur.height {
		height = c.cur.height - y
	}
	if width <= 0 || height <= 0 {
		return
	}
	c.dirty = append(c.dirty, Rect{x, y, width, height})
	c.dirty = mergeDirty(c.dirty)
}

func mergeDirty(rs []Rect) []Rect {
	for {
		merged := false
		for i := 0; i < len(rs) && !merged; i++ {
			for j := i + 1; j < len(rs); j++ {
				if touch(rs[i], rs[j]) {
					rs[i] = unionRect(rs[i], rs[j])
					rs = append(rs[:j], rs[j+1:]...)
					merged = true
					break
				}
			}
		}
		if !merged {
			break
		}
	}
	for len(rs) > 8 {
		bestI, bestJ, bestArea := 0, 1, maxInt
		for i := 0; i < len(rs); i++ {
			for j := i + 1; j < len(rs); j++ {
				u := unionRect(rs[i], rs[j])
				if a := u.Width * u.Height; a < bestArea {
					bestI, bestJ, bestArea = i, j, a
				}
			}
		}
		rs[bestI] = unionRect(rs[bestI], rs[bestJ])
		rs = append(rs[:bestJ], rs[bestJ+1:]...)
	}
	return rs
}

func touch(a, b Rect) bool {
	return a.X <= b.X+b.Width && b.X <= a.X+a.Width &&
		a.Y <= b.Y+b.Height && b.Y <= a.Y+a.Height
}

func unionRect(a, b Rect) Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.Width, b.X+b.Width), max(a.Y+a.Height, b.Y+b.Height)
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

func (c *Canvas) clipDirty() {
	for i := range c.dirty {
		r := &c.dirty[i]
		if r.X < 0 {
			r.Width += r.X
			r.X = 0
		}
		if r.Y < 0 {
			r.Height += r.Y
			r.Y = 0
		}
		if r.X+r.Width > c.cur.width {
			r.Width = c.cur.width - r.X
		}
		if r.Y+r.Height > c.cur.height {
			r.Height = c.cur.height - r.Y
		}
	}
	out := c.dirty[:0]
	for _, r := range c.dirty {
		if r.Width > 0 && r.Height > 0 {
			out = append(out, r)
		}
	}
	c.dirty = out
}
