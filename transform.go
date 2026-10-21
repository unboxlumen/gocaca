// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

// Invert swaps foreground and background colour components and toggles
// style bits, exactly like libcaca's caca_invert() (attr ^= 0x000f000f).
func (c *Canvas) Invert() {
	for i := range c.cur.attrs {
		c.cur.attrs[i] ^= 0x000f000f
	}
	c.touchAll()
}

// Flip reverses rows (top to bottom).
func (c *Canvas) Flip() {
	w, h := c.cur.width, c.cur.height
	for y := 0; y < h/2; y++ {
		y2 := h - 1 - y
		ch := append([]rune(nil), c.cur.chars[y*w:(y+1)*w]...)
		at := append([]Attr(nil), c.cur.attrs[y*w:(y+1)*w]...)
		copy(c.cur.chars[y*w:(y+1)*w], c.cur.chars[y2*w:(y2+1)*w])
		copy(c.cur.attrs[y*w:(y+1)*w], c.cur.attrs[y2*w:(y2+1)*w])
		copy(c.cur.chars[y2*w:(y2+1)*w], ch)
		copy(c.cur.attrs[y2*w:(y2+1)*w], at)
	}
	c.touchAll()
}

// Flop reverses columns (left to right).
func (c *Canvas) Flop() {
	w, h := c.cur.width, c.cur.height
	for y := 0; y < h; y++ {
		rowC := c.cur.chars[y*w : (y+1)*w]
		rowA := c.cur.attrs[y*w : (y+1)*w]
		for i, j := 0, w-1; i < j; i, j = i+1, j-1 {
			rowC[i], rowC[j] = rowC[j], rowC[i]
			rowA[i], rowA[j] = rowA[j], rowA[i]
		}
	}
	c.touchAll()
}

// Rotate180 rotates the current frame by 180 degrees.
func (c *Canvas) Rotate180() {
	w, h := c.cur.width, c.cur.height
	ch := make([]rune, len(c.cur.chars))
	at := make([]Attr, len(c.cur.attrs))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			di := (h-1-y)*w + (w - 1 - x)
			ch[di] = c.cur.chars[y*w+x]
			at[di] = c.cur.attrs[y*w+x]
		}
	}
	c.cur.chars, c.cur.attrs = ch, at
	c.touchAll()
}

// RotateLeft rotates the current frame 90 degrees counter-clockwise.
func (c *Canvas) RotateLeft() { c.rotate(true) }

// RotateRight rotates the current frame 90 degrees clockwise.
func (c *Canvas) RotateRight() { c.rotate(false) }

func (c *Canvas) rotate(left bool) {
	w, h := c.cur.width, c.cur.height
	nw, nh := h, w
	ch := make([]rune, nw*nh)
	at := make([]Attr, nw*nh)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			if left {
				nx, ny = y, w-1-x
			} else {
				nx, ny = h-1-y, x
			}
			ch[ny*nw+nx] = c.cur.chars[y*w+x]
			at[ny*nw+nx] = c.cur.attrs[y*w+x]
		}
	}
	c.cur.width, c.cur.height = nw, nh
	c.cur.chars, c.cur.attrs = ch, at
	c.touchAll()
}

// StretchHorizontal doubles the width by duplicating each column.
func (c *Canvas) StretchHorizontal() {
	w, h := c.cur.width, c.cur.height
	ch := make([]rune, w*2*h)
	at := make([]Attr, w*2*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			di := y*w*2 + x*2
			ch[di], ch[di+1] = c.cur.chars[y*w+x], c.cur.chars[y*w+x]
			at[di], at[di+1] = c.cur.attrs[y*w+x], c.cur.attrs[y*w+x]
		}
	}
	c.cur.width *= 2
	c.cur.chars, c.cur.attrs = ch, at
	c.touchAll()
}

// StretchVertical doubles the height by duplicating each row.
func (c *Canvas) StretchVertical() {
	w, h := c.cur.width, c.cur.height
	ch := make([]rune, w*h*2)
	at := make([]Attr, w*h*2)
	for y := 0; y < h; y++ {
		copy(ch[(2*y)*w:(2*y+1)*w], c.cur.chars[y*w:(y+1)*w])
		copy(ch[(2*y+1)*w:(2*y+2)*w], c.cur.chars[y*w:(y+1)*w])
		copy(at[(2*y)*w:(2*y+1)*w], c.cur.attrs[y*w:(y+1)*w])
		copy(at[(2*y+1)*w:(2*y+2)*w], c.cur.attrs[y*w:(y+1)*w])
	}
	c.cur.height *= 2
	c.cur.chars, c.cur.attrs = ch, at
	c.touchAll()
}

func (c *Canvas) touchAll() {
	c.normalizeFullwidth()
	c.dirty = c.dirty[:0]
	if !c.dirtyDisabled() {
		c.addDirtyRect(0, 0, c.cur.width, c.cur.height)
	}
}

// normalizeFullwidth removes orphaned fullwidth markers left behind by
// geometric transforms. A fullwidth glyph is only valid immediately before
// its Fullwidth marker.
func (c *Canvas) normalizeFullwidth() {
	w, h := c.cur.width, c.cur.height
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			ch := c.cur.chars[i]
			if ch == Fullwidth {
				if x == 0 || !IsFullwidth(c.cur.chars[i-1]) {
					c.cur.chars[i] = ' '
				}
				continue
			}
			if IsFullwidth(ch) && (x+1 >= w || c.cur.chars[i+1] != Fullwidth) {
				c.cur.chars[i] = ' '
			}
		}
	}
}
