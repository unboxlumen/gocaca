// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ANSI SGR colour order is black, red, green, yellow, blue, magenta, cyan,
// white. The Color enum uses the DOS/BGI order, so the two need mapping.
var ansiToColor = [8]Color{
	Black, Red, Green, Brown, Blue, Magenta, Cyan, LightGray,
}

var ansiBrightToColor = [8]Color{
	DarkGray, LightRed, LightGreen, Yellow, LightBlue, LightMagenta,
	LightCyan, White,
}

func colorToANSIIndex(c Color) int {
	for i, v := range ansiToColor {
		if v == c {
			return i
		}
	}
	for i, v := range ansiBrightToColor {
		if v == c {
			return i
		}
	}
	return -1
}

func ansiFgCode(c Color) string {
	if c == Default || c == Transparent {
		return "39"
	}
	if c < 8 {
		return strconv.Itoa(30 + colorToANSIIndex(c))
	}
	if c < 16 {
		return strconv.Itoa(90 + colorToANSIIndex(c))
	}
	return "39"
}

func ansiBgCode(c Color) string {
	if c == Default || c == Transparent {
		return "49"
	}
	if c < 8 {
		return strconv.Itoa(40 + colorToANSIIndex(c))
	}
	if c < 16 {
		return strconv.Itoa(100 + colorToANSIIndex(c))
	}
	return "49"
}

func sgr(a Attr) string {
	codes := []string{"0"}
	s := a.Style()
	if s&Bold != 0 {
		codes = append(codes, "1")
	}
	if s&Italics != 0 {
		codes = append(codes, "3")
	}
	if s&Underline != 0 {
		codes = append(codes, "4")
	}
	if s&Blink != 0 {
		codes = append(codes, "5")
	}
	fg14, bg14 := a.fg14(), a.bg14()
	if isANSIMarker(fg14) {
		codes = append(codes, ansiFgCode(a.FG()))
	} else {
		rgb := a.RGB12FG()
		codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d",
			(rgb>>8&0xf)*17, (rgb>>4&0xf)*17, (rgb&0xf)*17))
	}
	if isANSIMarker(bg14) {
		codes = append(codes, ansiBgCode(a.BG()))
	} else {
		rgb := a.RGB12BG()
		codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d",
			(rgb>>8&0xf)*17, (rgb>>4&0xf)*17, (rgb&0xf)*17))
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

func (c *Canvas) encodeANSI(utf8Mode bool) []byte {
	var b strings.Builder
	b.WriteString("\x1b[0m")
	for y := 0; y < c.cur.height; y++ {
		last := Attr(^uint32(0))
		have := false
		for x := 0; x < c.cur.width; x++ {
			i := y*c.cur.width + x
			ch := c.cur.chars[i]
			if ch == Fullwidth {
				continue
			}
			a := c.cur.attrs[i]
			if !have || a != last {
				b.WriteString(sgr(a))
				last, have = a, true
			}
			if utf8Mode {
				b.WriteRune(ch)
			} else {
				b.WriteByte(UTF32ToCP437(ch))
			}
		}
		b.WriteString("\x1b[0m\n")
	}
	return []byte(b.String())
}

type ansiState struct {
	fg, bg       Color
	fgRGB, bgRGB bool
	fgVal, bgVal uint32
	style        Style
}

func newANSIState() *ansiState {
	return &ansiState{fg: Default, bg: Transparent}
}

func (s *ansiState) reset() {
	*s = ansiState{fg: Default, bg: Transparent}
}

func (s *ansiState) attr() Attr {
	if s.fgRGB || s.bgRGB {
		fg := uint16(s.fgVal)
		bg := uint16(s.bgVal)
		if !s.fgRGB {
			fg = ansiToRGB14(s.fg)
		}
		if !s.bgRGB {
			bg = ansiToRGB14(s.bg)
		}
		return Attr(uint32(bg)<<18 | uint32(fg)<<4 | uint32(s.style)&0xf)
	}
	return ANSIAttr(s.fg, s.bg, s.style)
}

// ansiToRGB14 converts a palette index to libcaca's 14-bit colour value.
func ansiToRGB14(c Color) uint16 {
	if c >= 16 {
		return uint16(ansitab14[Black])
	}
	return ansitab14[c]
}

func (s *ansiState) set256(n int, fg bool) {
	if n < 0 || n > 255 {
		return
	}
	var r, g, b int
	switch {
	case n < 16:
		if fg {
			s.fg, s.fgRGB = Color(n), false
		} else {
			s.bg, s.bgRGB = Color(n), false
		}
		return
	case n < 232:
		n -= 16
		levels := [6]int{0, 95, 135, 175, 215, 255}
		r, g, b = levels[n/36], levels[(n/6)%6], levels[n%6]
	default:
		v := 8 + (n-232)*10
		r, g, b = v, v, v
	}
	v := uint32(argb16To14(uint16(0xf000 | (r>>4)<<8 | (g>>4)<<4 | (b >> 4))))
	if fg {
		s.fgRGB = true
		s.fgVal = v
	} else {
		s.bgRGB = true
		s.bgVal = v
	}
}

func (s *ansiState) applySGR(params string) {
	if params == "" {
		s.reset()
		return
	}
	parts := strings.Split(params, ";")
	nums := make([]int, len(parts))
	for i, p := range parts {
		if p == "" {
			nums[i] = 0
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			nums[i] = -1
		} else {
			nums[i] = v
		}
	}
	for i := 0; i < len(nums); i++ {
		n := nums[i]
		switch {
		case n == 0:
			s.reset()
		case n == 1:
			s.style |= Bold
		case n == 3:
			s.style |= Italics
		case n == 4:
			s.style |= Underline
		case n == 5:
			s.style |= Blink
		case n == 22:
			s.style &^= Bold
		case n == 23:
			s.style &^= Italics
		case n == 24:
			s.style &^= Underline
		case n == 25:
			s.style &^= Blink
		case n >= 30 && n <= 37:
			s.fg, s.fgRGB = ansiToColor[n-30], false
		case n == 39:
			s.fg, s.fgRGB = Default, false
		case n >= 40 && n <= 47:
			s.bg, s.bgRGB = ansiToColor[n-40], false
		case n == 49:
			s.bg, s.bgRGB = Default, false
		case n >= 90 && n <= 97:
			s.fg, s.fgRGB = ansiBrightToColor[n-90], false
		case n >= 100 && n <= 107:
			s.bg, s.bgRGB = ansiBrightToColor[n-100], false
		case n == 38 && i+1 < len(nums):
			switch nums[i+1] {
			case 2:
				if i+4 < len(nums) {
					s.fgRGB = true
					s.fgVal = uint32(argb16To14(uint16(0xf000 |
						rgb12(nums[i+2], nums[i+3], nums[i+4]))))
					i += 4
				}
			case 5:
				if i+2 < len(nums) {
					s.set256(nums[i+2], true)
					i += 2
				}
			}
		case n == 48 && i+1 < len(nums):
			switch nums[i+1] {
			case 2:
				if i+4 < len(nums) {
					s.bgRGB = true
					s.bgVal = uint32(argb16To14(uint16(0xf000 |
						rgb12(nums[i+2], nums[i+3], nums[i+4]))))
					i += 4
				}
			case 5:
				if i+2 < len(nums) {
					s.set256(nums[i+2], false)
					i += 2
				}
			}
		}
	}
}

func rgb12(r, g, b int) uint32 {
	clamp := func(v int) uint32 {
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint32(v >> 4)
	}
	return clamp(r)<<8 | clamp(g)<<4 | clamp(b)
}

func decodeANSI(data []byte, utf8Mode bool) (*Canvas, error) {
	st := newANSIState()
	rows := make([][]importCell, 0, 8)
	lineCount, maxW := 0, 0
	x, y := 0, 0

	put := func(ch rune, attr Attr) {
		w := 1
		if IsFullwidth(ch) {
			w = 2
		}
		for len(rows) <= y {
			rows = append(rows, nil)
		}
		row := &rows[y]
		for len(*row) < x+w {
			*row = append(*row, importCell{' ', attr})
		}
		if x >= 0 {
			(*row)[x] = importCell{ch, attr}
			if w == 2 {
				(*row)[x+1] = importCell{Fullwidth, attr}
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

	for i := 0; i < len(data); {
		b := data[i]
		if b == 0x1b {
			if i+1 < len(data) && data[i+1] == '[' {
				j := i + 2
				for j < len(data) && !(data[j] >= 0x40 && data[j] <= 0x7e) {
					j++
				}
				if j >= len(data) {
					break
				}
				if data[j] == 'm' {
					st.applySGR(string(data[i+2 : j]))
				}
				i = j + 1
				continue
			}
			i += 2
			continue
		}
		switch b {
		case '\n':
			y++
			x = 0
			if y > lineCount {
				lineCount = y
			}
			i++
			continue
		case '\r':
			x = 0
			i++
			continue
		case '\t':
			x = (x + 8) &^ 7
			i++
			continue
		}
		if b < 0x20 {
			i++
			continue
		}
		if utf8Mode {
			r, size := utf8.DecodeRune(data[i:])
			if size == 0 {
				break
			}
			put(r, st.attr())
			i += size
		} else if b < 0x80 {
			put(rune(b), st.attr())
			i++
		} else {
			put(CP437ToUTF32(b), st.attr())
			i++
		}
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
