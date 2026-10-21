// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to
// the extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import stdcolor "image/color"

// Color is an ANSI colour index or one of the special values Default and
// Transparent. The values follow the DOS/BGI order used by libcaca.
type Color uint8

const (
	Black Color = iota
	Blue
	Green
	Cyan
	Red
	Magenta
	Brown
	LightGray
	DarkGray
	LightBlue
	LightGreen
	LightCyan
	LightRed
	LightMagenta
	Yellow
	White
	Default
	Transparent Color = 0x20
)

// Style is a bit mask of text styles.
type Style uint8

const (
	Bold Style = 1 << iota
	Italics
	Underline
	Blink
)

// Attr is a 32-bit cell attribute, bit-compatible with libcaca.
//
//	bits 0..3    style
//	bits 4..17   foreground (14 bits)
//	bits 18..31  background (14 bits)
//
// An ANSI component is stored as index|0x40 and is below 0x50. The values
// 0x50 and 0x60 mean Default and Transparent. Anything else is a 14-bit
// ARGB value: alpha in bits 11..13, 12-bit RGB in bits 0..10.
type Attr uint32

const (
	attrStyleMask Attr = 0x0000000f
	attrFGMask    Attr = 0x0003fff0
	attrBGMask    Attr = 0xfffc0000

	ansiMarker      Attr = 0x40
	ansiDefault     Attr = 0x10 | ansiMarker
	ansiTransparent Attr = 0x20 | ansiMarker
)

// ansitab16 is libcaca's terminal palette, packed as 0xARGB with four bits
// per component.
var ansitab16 = [16]uint16{
	0xf000, 0xf00a, 0xf0a0, 0xf0aa, 0xfa00, 0xfa0a, 0xfa50, 0xfaaa,
	0xf555, 0xf55f, 0xf5f5, 0xf5ff, 0xff55, 0xff5f, 0xfff5, 0xffff,
}

// ansitab14 is the same palette on 14 bits (3-4-4-3).
var ansitab14 = [16]uint16{
	0x3800, 0x3805, 0x3850, 0x3855, 0x3d00, 0x3d05, 0x3d28, 0x3d55,
	0x3aaa, 0x3aaf, 0x3afa, 0x3aff, 0x3faa, 0x3faf, 0x3ffa, 0x3fff,
}

// ANSIAttr builds an attribute from ANSI colours and a style mask.
// Like libcaca's caca_set_color_ansi(), values above 0x20 are rejected by
// SetColor; this constructor stores whatever it is given.
func ANSIAttr(fg, bg Color, style Style) Attr {
	return Attr((uint32(bg)|0x40)<<18 | (uint32(fg)|0x40)<<4 | uint32(style)&0xf)
}

// RGBAttr builds an attribute from 8-bit RGB colours and a style mask.
// Each channel is quantised to four bits, matching libcaca's ARGB4444
// encoding. The alpha channel is honoured.
func RGBAttr(fg, bg stdcolor.RGBA, style Style) Attr {
	return ARGB16Attr(argb16(fg), argb16(bg), style)
}

// ARGB16Attr builds an attribute from two 16-bit ARGB4444 values, exactly
// like libcaca's caca_set_color_argb().
func ARGB16Attr(fg, bg uint16, style Style) Attr {
	return attrFromARGB16(fg, bg, style)
}

func argb16(c stdcolor.RGBA) uint16 {
	return uint16(c.A>>4)<<12 | uint16(c.R>>4)<<8 | uint16(c.G>>4)<<4 | uint16(c.B>>4)
}

// argb16To14 reproduces libcaca's caca_set_color_argb() packing.
func argb16To14(v uint16) uint32 {
	if v < 0x100 {
		v += 0x100
	}
	return uint32((v>>1)&0x7ff) | uint32(v>>13)<<11
}

func attrFromARGB16(fg, bg uint16, style Style) Attr {
	fg14 := argb16To14(fg)
	bg14 := argb16To14(bg)
	return Attr(bg14<<18 | fg14<<4 | uint32(style)&0xf)
}

func (a Attr) fg14() uint16 { return uint16((uint32(a) >> 4) & 0x3fff) }
func (a Attr) bg14() uint16 { return uint16(uint32(a) >> 18) }

// Style returns the style bits.
func (a Attr) Style() Style { return Style(a & attrStyleMask) }

// WithStyle returns a copy of a with the style bits replaced.
func (a Attr) WithStyle(s Style) Attr {
	return (a &^ attrStyleMask) | Attr(s)&attrStyleMask
}

// IsRGB reports whether either component is stored as a truecolour value.
func (a Attr) IsRGB() bool { return !isANSIMarker(a.fg14()) || !isANSIMarker(a.bg14()) }

func isANSIMarker(v uint16) bool { return v >= 0x40 && v < ansiDefault16 }

const (
	ansiDefault16     uint16 = 0x50
	ansiTransparent16 uint16 = 0x60
)

// FG returns the ANSI foreground colour, choosing the nearest palette entry
// for truecolour values. It may return Default or Transparent.
func (a Attr) FG() Color { return nearestANSI16(a.fg14()) }

// BG returns the ANSI background colour, choosing the nearest palette entry
// for truecolour values. It may return Default or Transparent.
func (a Attr) BG() Color { return nearestANSI16(a.bg14()) }

// ANSI returns the packed DOS colour byte: background in the high nibble,
// foreground in the low nibble. Special colours become LightGray/Black.
func (a Attr) ANSI() uint8 {
	fg := a.FG()
	bg := a.BG()
	if fg >= 0x10 {
		fg = LightGray
	}
	if bg >= 0x10 {
		bg = Black
	}
	return uint8(fg) | uint8(bg)<<4
}

// RGB returns the foreground and background as 8-bit RGB colours. ANSI
// components are expanded through the terminal palette. ok reports whether
// either component is stored as a truecolour value.
func (a Attr) RGB() (fg, bg stdcolor.RGBA, ok bool) {
	return rgb12ToRGBA(a.RGB12FG()), rgb12ToRGBA(a.RGB12BG()), a.IsRGB()
}

// RGB12FG returns the 12-bit foreground colour, mirroring libcaca's
// caca_attr_to_rgb12_fg().
func (a Attr) RGB12FG() uint16 {
	fg := a.fg14()
	if isANSIMarker(fg) {
		return ansitab16[fg^0x40] & 0x0fff
	}
	if fg == ansiDefault16 || fg == ansiTransparent16 {
		return ansitab16[LightGray] & 0x0fff
	}
	return (fg << 1) & 0x0fff
}

// RGB12BG returns the 12-bit background colour, mirroring libcaca's
// caca_attr_to_rgb12_bg().
func (a Attr) RGB12BG() uint16 {
	bg := a.bg14()
	if isANSIMarker(bg) {
		return ansitab16[bg^0x40] & 0x0fff
	}
	if bg == ansiDefault16 || bg == ansiTransparent16 {
		return ansitab16[Black] & 0x0fff
	}
	return (bg << 1) & 0x0fff
}

// ARGB64 returns the eight 4-bit components in libcaca's order: background
// alpha, red, green, blue, then foreground alpha, red, green, blue.
func (a Attr) ARGB64() [8]byte {
	fg := a.fg14()
	bg := a.bg14()

	switch {
	case isANSIMarker(bg):
		bg = ansitab16[bg^0x40]
	case bg == ansiDefault16:
		bg = ansitab16[Black]
	case bg == ansiTransparent16:
		bg = 0x0fff
	default:
		bg = (bg << 2 & 0xf000) | (bg << 1 & 0x0fff)
	}

	switch {
	case isANSIMarker(fg):
		fg = ansitab16[fg^0x40]
	case fg == ansiDefault16:
		fg = ansitab16[LightGray]
	case fg == ansiTransparent16:
		fg = 0x0fff
	default:
		fg = (fg << 2 & 0xf000) | (fg << 1 & 0x0fff)
	}

	return [8]byte{
		byte(bg >> 12), byte(bg >> 8 & 0xf), byte(bg >> 4 & 0xf), byte(bg & 0xf),
		byte(fg >> 12), byte(fg >> 8 & 0xf), byte(fg >> 4 & 0xf), byte(fg & 0xf),
	}
}

// nearestANSI16 mirrors libcaca's nearest_ansi().
func nearestANSI16(argb14 uint16) Color {
	if isANSIMarker(argb14) {
		return Color(argb14 ^ 0x40)
	}
	if argb14 == ansiDefault16 || argb14 == ansiTransparent16 {
		return Color(argb14 ^ 0x40)
	}
	if argb14 < 0x0fff {
		return Transparent
	}

	best := Default
	dist := uint32(0x3fff)
	for i := 0; i < 16; i++ {
		var d uint32
		a := uint32(ansitab14[i]>>7) & 0xf
		b := uint32(argb14>>7) & 0xf
		d += (a - b) * (a - b)
		a = uint32(ansitab14[i]>>3) & 0xf
		b = uint32(argb14>>3) & 0xf
		d += (a - b) * (a - b)
		a = uint32(ansitab14[i]<<1) & 0xf
		b = uint32(argb14<<1) & 0xf
		d += (a - b) * (a - b)
		if d < dist {
			dist = d
			best = Color(i)
		}
	}
	return best
}

func rgb12ToRGBA(v uint16) stdcolor.RGBA {
	return stdcolor.RGBA{
		R: uint8(v>>8&0xf) * 17,
		G: uint8(v>>4&0xf) * 17,
		B: uint8(v&0xf) * 17,
		A: 0xff,
	}
}

// ansiColorRGBA expands a palette index to 8-bit RGB.
func ansiColorRGBA(c Color) stdcolor.RGBA {
	if c >= 16 {
		return stdcolor.RGBA{}
	}
	v := ansitab16[c]
	return stdcolor.RGBA{
		R: uint8(v>>8&0xf) * 17,
		G: uint8(v>>4&0xf) * 17,
		B: uint8(v&0xf) * 17,
		A: 0xff,
	}
}

func validColor(c Color) bool { return c <= 0x20 }
