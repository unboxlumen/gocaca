// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"strings"
	"unicode/utf8"
)

type importCell struct {
	ch   rune
	attr Attr
}

func (c *Canvas) encodeText() []byte {
	var b strings.Builder
	for y := 0; y < c.cur.height; y++ {
		for x := 0; x < c.cur.width; x++ {
			ch := c.cur.chars[y*c.cur.width+x]
			if ch == Fullwidth {
				continue
			}
			b.WriteRune(ch)
		}
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func decodeText(data []byte) (*Canvas, error) {
	// Text import uses CACA_DEFAULT on CACA_TRANSPARENT in libcaca.
	def := ANSIAttr(Default, Transparent, 0)
	rows := make([][]importCell, 0, 8)
	lineCount := 0
	maxW := 0
	x, y := 0, 0

	put := func(ch rune) {
		w := 1
		if IsFullwidth(ch) {
			w = 2
		}
		for len(rows) <= y {
			rows = append(rows, nil)
		}
		row := &rows[y]
		for len(*row) < x+w {
			*row = append(*row, importCell{' ', def})
		}
		if x >= 0 {
			(*row)[x] = importCell{ch, def}
			if w == 2 {
				(*row)[x+1] = importCell{Fullwidth, def}
			}
		}
		x += w
		if x > maxW {
			maxW = x
		}
		if y+1 > lineCount {
			lineCount = y + 1
		}
	}

	s := string(data)
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if size == 0 {
			break
		}
		s = s[size:]
		switch r {
		case '\n':
			y++
			x = 0
			if y > lineCount {
				lineCount = y
			}
		case '\r':
			x = 0
		case '\t':
			x = (x + 8) &^ 7
		default:
			put(r)
		}
	}
	if lineCount == 0 && len(rows) > 0 {
		lineCount = len(rows)
	}
	nc, err := NewCanvas(maxW, lineCount)
	if err != nil {
		return nil, err
	}
	for yy, row := range rows {
		for xx, cell := range row {
			if cell.ch == Fullwidth {
				continue
			}
			nc.PutChar(xx, yy, cell.ch)
			nc.PutAttr(xx, yy, cell.attr)
		}
	}
	return nc, nil
}
