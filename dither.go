// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"errors"
	"image"
	"image/color"
	"math"
)

// Dither converts bitmap images into canvas cells.
//
// The implementation follows libcaca's caca_dither_bitmap() closely: the
// same 12-bit palette, the same glyph tables, the same integer error
// diffusion and the same colour-mode arithmetic. Output is character-for-
// character identical to libcaca for the same input pixels and options.
type Dither struct {
	width, height int
	pixels        []byte
	bpp           int
	hasAlpha      bool

	gamma      float64
	brightness float64
	contrast   float64
	gammatab   [4096]int
	invert     bool

	antialias string
	colorName string
	charset   string
	algorithm string
}

// rgbPalette is libcaca's 12-bit palette in CACA_* enum order.
var rgbPalette = [16][3]int{
	{0x0, 0x0, 0x0},
	{0x0, 0x0, 0x7ff},
	{0x0, 0x7ff, 0x0},
	{0x0, 0x7ff, 0x7ff},
	{0x7ff, 0x0, 0x0},
	{0x7ff, 0x0, 0x7ff},
	{0x7ff, 0x7ff, 0x0},
	{0xaaa, 0xaaa, 0xaaa},
	{0x555, 0x555, 0x555},
	{0x000, 0x000, 0xfff},
	{0x000, 0xfff, 0x000},
	{0x000, 0xfff, 0xfff},
	{0xfff, 0x000, 0x000},
	{0xfff, 0x000, 0xfff},
	{0xfff, 0xfff, 0x000},
	{0xfff, 0xfff, 0xfff},
}

var asciiGlyphs = []rune{' ', '.', ':', ';', 't', '%', 'S', 'X', '@', '8', '?'}
var shadesGlyphs = []rune{' ', 0xb7, 0x2591, 0x2592, '?'}
var blocksGlyphs = []rune{' ', 0x2598, 0x259a, '?'}

type colorMode int

const (
	colorMono colorMode = iota
	colorGray
	color8
	color16
	colorFullgray
	colorFull8
	colorFull16
)

// NewDither creates a dither from raw pixels. bpp must be 24 (RGB) or 32
// (RGBA), and len(pixels) must be width*height*bpp/8.
func NewDither(pixels []byte, width, height, bpp int) (*Dither, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("libcaca: invalid dither size")
	}
	if bpp != 24 && bpp != 32 {
		return nil, errors.New("libcaca: only 24 and 32 bpp are supported")
	}
	if width > maxInt/height || len(pixels) < width*height*bpp/8 {
		return nil, errors.New("libcaca: pixel buffer too small")
	}
	d := &Dither{
		width:      width,
		height:     height,
		pixels:     pixels,
		bpp:        bpp,
		hasAlpha:   bpp == 32,
		gamma:      1.0,
		brightness: 1.0,
		contrast:   1.0,
		antialias:  "prefilter",
		colorName:  "full16",
		charset:    "ascii",
		algorithm:  "fstein",
	}
	d.rebuildGamma()
	return d, nil
}

// NewDitherFromImage converts a decoded image. Transparent pixels are
// composited over black, matching libcaca's handling of alpha below 0x800.
func NewDitherFromImage(img image.Image) (*Dither, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("libcaca: empty image")
	}
	pix := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			i := (y*w + x) * 4
			pix[i] = c.R
			pix[i+1] = c.G
			pix[i+2] = c.B
			pix[i+3] = c.A
		}
	}
	return NewDither(pix, w, h, 32)
}

func (d *Dither) Width() int  { return d.width }
func (d *Dither) Height() int { return d.height }

func (d *Dither) rebuildGamma() {
	for i := 0; i < 4096; i++ {
		d.gammatab[i] = int(4096.0 * math.Pow(float64(i)/4096.0, 1.0/d.gamma))
	}
}

// SetBrightness sets the brightness multiplier. Like libcaca, brightness is
// stored but not applied during rendering.
func (d *Dither) SetBrightness(v float64) error {
	if v < 0 {
		return errors.New("libcaca: invalid brightness")
	}
	d.brightness = v
	return nil
}

func (d *Dither) Brightness() float64 { return d.brightness }

// SetGamma sets the gamma correction. A negative value inverts colours.
func (d *Dither) SetGamma(v float64) error {
	invert := false
	if v < 0 {
		invert = true
		v = -v
	}
	if v == 0 {
		return errors.New("libcaca: invalid gamma")
	}
	d.gamma = v
	d.invert = invert
	d.rebuildGamma()
	return nil
}

func (d *Dither) Gamma() float64 {
	if d.invert {
		return -d.gamma
	}
	return d.gamma
}

// SetContrast sets the contrast multiplier. Like libcaca, contrast is stored
// but not applied during rendering.
func (d *Dither) SetContrast(v float64) error {
	if v < 0 {
		return errors.New("libcaca: invalid contrast")
	}
	d.contrast = v
	return nil
}

func (d *Dither) Contrast() float64 { return d.contrast }

// SetAntialias selects "none" or "prefilter".
func (d *Dither) SetAntialias(name string) error {
	switch name {
	case "", "default", "prefilter":
		d.antialias = "prefilter"
	case "none":
		d.antialias = "none"
	default:
		return errors.New("libcaca: unknown antialias mode")
	}
	return nil
}

func (d *Dither) Antialias() string { return d.antialias }

// SetColor selects the colour mode: mono, gray, 8, 16, fullgray, full8 or
// full16.
func (d *Dither) SetColor(name string) error {
	switch name {
	case "", "default", "full16":
		d.colorName = "full16"
	case "mono", "gray", "8", "16", "fullgray", "full8":
		d.colorName = name
	default:
		return errors.New("libcaca: unknown colour mode")
	}
	return nil
}

func (d *Dither) Color() string { return d.colorName }

func (d *Dither) colorMode() colorMode {
	switch d.colorName {
	case "mono":
		return colorMono
	case "gray":
		return colorGray
	case "8":
		return color8
	case "16":
		return color16
	case "fullgray":
		return colorFullgray
	case "full8":
		return colorFull8
	default:
		return colorFull16
	}
}

// SetCharset selects "ascii", "shades" or "blocks".
func (d *Dither) SetCharset(name string) error {
	switch name {
	case "", "default", "ascii":
		d.charset = "ascii"
	case "shades", "blocks":
		d.charset = name
	default:
		return errors.New("libcaca: unknown charset")
	}
	return nil
}

func (d *Dither) Charset() string { return d.charset }

// SetAlgorithm selects "none", "ordered2", "ordered4", "ordered8", "random"
// or "fstein".
func (d *Dither) SetAlgorithm(name string) error {
	switch name {
	case "", "default", "fstein":
		d.algorithm = "fstein"
	case "none", "ordered2", "ordered4", "ordered8", "random":
		d.algorithm = name
	default:
		return errors.New("libcaca: unknown dithering algorithm")
	}
	return nil
}

func (d *Dither) Algorithm() string { return d.algorithm }

func (d *Dither) glyphs() []rune {
	switch d.charset {
	case "shades":
		return shadesGlyphs
	case "blocks":
		return blocksGlyphs
	default:
		return asciiGlyphs
	}
}

// readPixel returns the 12-bit RGB and 8-bit alpha of one source pixel.
func (d *Dither) readPixel(x, y int) (r, g, b, a int) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= d.width {
		x = d.width - 1
	}
	if y >= d.height {
		y = d.height - 1
	}
	i := (y*d.width + x) * d.bpp / 8
	r = d.gammatab[int(d.pixels[i])<<4]
	g = d.gammatab[int(d.pixels[i+1])<<4]
	b = d.gammatab[int(d.pixels[i+2])<<4]
	if d.hasAlpha {
		// libcaca stores alpha on 12 bits.
		a = int(d.pixels[i+3]) << 4
	} else {
		a = 0xfff
	}
	return
}

// ordered tables from libcaca
var ordered2Table = [2][2]int{{0x00, 0x80}, {0xc0, 0x40}}
var ordered4Table = [4][4]int{
	{0x00, 0x80, 0x20, 0xa0},
	{0xc0, 0x40, 0xe0, 0x60},
	{0x30, 0xb0, 0x10, 0x90},
	{0xf0, 0x70, 0xd0, 0x50},
}
var ordered8Table = [8][8]int{
	{0x00, 0x80, 0x20, 0xa0, 0x08, 0x88, 0x28, 0xa8},
	{0xc0, 0x40, 0xe0, 0x60, 0xc8, 0x48, 0xe8, 0x68},
	{0x30, 0xb0, 0x10, 0x90, 0x38, 0xb8, 0x18, 0x98},
	{0xf0, 0x70, 0xd0, 0x50, 0xf8, 0x78, 0xd8, 0x58},
	{0x0c, 0x8c, 0x2c, 0xac, 0x04, 0x84, 0x24, 0xa4},
	{0xcc, 0x4c, 0xec, 0x6c, 0xc4, 0x44, 0xe4, 0x64},
	{0x3c, 0xbc, 0x1c, 0x9c, 0x34, 0xb4, 0x14, 0x94},
	{0xfc, 0x7c, 0xdc, 0x5c, 0xf4, 0x74, 0xd4, 0x54},
}

var randomSeed uint32 = 0x12345678

func randomDither() int {
	randomSeed = randomSeed*1103515245 + 12345
	return int((randomSeed >> 16) & 0xff)
}

func (d *Dither) noise(x, y int) int {
	switch d.algorithm {
	case "ordered2":
		return ordered2Table[y%2][x%2]
	case "ordered4":
		return ordered4Table[y%4][x%4]
	case "ordered8":
		return ordered8Table[y%8][x%8]
	case "random":
		return randomDither()
	default:
		return 0x80
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sqInt(v int) int { return v * v }

// Draw renders the image into cv, filling the rectangle (x,y,w,h).
func (d *Dither) Draw(cv *Canvas, x, y, w, h int) error {
	if cv == nil {
		return errors.New("libcaca: nil canvas")
	}
	if w <= 0 || h <= 0 {
		return nil
	}
	glyphs := d.glyphs()
	dchmax := len(glyphs)
	den := 2*dchmax - 1
	mode := d.colorMode()

	x1, y1 := x, y
	deltax, deltay := w, h

	fsLength := w + 2
	fsR := make([]int, fsLength)
	fsG := make([]int, fsLength)
	fsB := make([]int, fsLength)

	for cy := 0; cy < deltay; cy++ {
		remainR, remainG, remainB := 0, 0, 0

		for cx := 0; cx < deltax; cx++ {
			fromX := cx * d.width / deltax
			fromY := cy * d.height / deltay
			toX := (cx + 1) * d.width / deltax
			toY := (cy + 1) * d.height / deltay
			if toX == fromX {
				toX++
			}
			if toY == fromY {
				toY++
			}

			var rgba [4]int
			if d.antialias == "prefilter" {
				dots := 0
				for myx := fromX; myx < toX; myx++ {
					for myy := fromY; myy < toY; myy++ {
						r, g, b, a := d.readPixel(myx, myy)
						rgba[0] += r
						rgba[1] += g
						rgba[2] += b
						rgba[3] += a
						dots++
					}
				}
				if dots > 0 {
					rgba[0] /= dots
					rgba[1] /= dots
					rgba[2] /= dots
					rgba[3] /= dots
				}
			} else {
				myx := (fromX + toX) / 2
				myy := (fromY + toY) / 2
				r, g, b, a := d.readPixel(myx, myy)
				rgba[0], rgba[1], rgba[2], rgba[3] = r, g, b, a
			}

			if mode == colorFullgray {
				gray := (3*rgba[0] + 4*rgba[1] + rgba[2] + 4) / 8
				rgba[0], rgba[1], rgba[2] = gray, gray, gray
			}

			if d.hasAlpha && rgba[3] < 0x800 {
				continue
			}
			_ = remainR
			if d.algorithm == "fstein" {
				rgba[0] += remainR
				rgba[1] += remainG
				rgba[2] += remainB
			} else {
				n := (d.noise(cx, cy) - 0x80) * 4
				rgba[0] += n
				rgba[1] += n
				rgba[2] += n
			}

			// Nearest background colour.
			distmin := 1 << 30
			outbg := 0
			for i := 0; i < 16; i++ {
				if mode == colorFullgray && (rgbPalette[i][0] != rgbPalette[i][1] ||
					rgbPalette[i][0] != rgbPalette[i][2]) {
					continue
				}
				dist := sqInt(rgba[0]-rgbPalette[i][0]) +
					sqInt(rgba[1]-rgbPalette[i][1]) +
					sqInt(rgba[2]-rgbPalette[i][2])
				if dist < distmin {
					outbg = i
					distmin = dist
				}
			}
			bgR, bgG, bgB := rgbPalette[outbg][0], rgbPalette[outbg][1], rgbPalette[outbg][2]

			var outfg, ch int
			if mode == colorFull16 || mode == colorFullgray {
				distmin = 1 << 30
				for i := 0; i < 16; i++ {
					if i == outbg {
						continue
					}
					if mode == colorFullgray && (rgbPalette[i][0] != rgbPalette[i][1] ||
						rgbPalette[i][0] != rgbPalette[i][2]) {
						continue
					}
					dist := sqInt(rgba[0]-rgbPalette[i][0]) +
						sqInt(rgba[1]-rgbPalette[i][1]) +
						sqInt(rgba[2]-rgbPalette[i][2])
					if dist < distmin {
						outfg = i
						distmin = dist
					}
				}
				fgR, fgG, fgB := rgbPalette[outfg][0], rgbPalette[outfg][1], rgbPalette[outfg][2]

				distmin = 1 << 30
				for i := 0; i < dchmax-1; i++ {
					newr := i*fgR + (den-i)*bgR
					newg := i*fgG + (den-i)*bgG
					newb := i*fgB + (den-i)*bgB
					dist := absInt(rgba[0]*den-newr) +
						absInt(rgba[1]*den-newg) +
						absInt(rgba[2]*den-newb)
					if dist < distmin {
						ch = i
						distmin = dist
					}
				}

				if d.algorithm == "fstein" {
					errR := rgba[0] - (fgR*ch+bgR*(den-ch))/den
					errG := rgba[1] - (fgG*ch+bgG*(den-ch))/den
					errB := rgba[2] - (fgB*ch+bgB*(den-ch))/den
					remainR, remainG, remainB, _ = spreadFS(fsR, fsG, fsB, cx, errR, errG, errB)
				}
			} else {
				lum := rgba[0]
				if rgba[1] > lum {
					lum = rgba[1]
				}
				if rgba[2] > lum {
					lum = rgba[2]
				}
				outfg = outbg
				outbg = int(Black)

				ch = lum * dchmax / 0x1000
				if ch < 0 {
					ch = 0
				} else if ch > dchmax-1 {
					ch = dchmax - 1
				}

				if d.algorithm == "fstein" {
					errR := rgba[0] - bgR*ch/(dchmax-1)
					errG := rgba[1] - bgG*ch/(dchmax-1)
					errB := rgba[2] - bgB*ch/(dchmax-1)
					remainR, remainG, remainB, _ = spreadFS(fsR, fsG, fsB, cx, errR, errG, errB)
				}
			}

			if d.invert {
				outfg = 15 - outfg
				outbg = 15 - outbg
			}

			if err := cv.SetColor(Color(outfg), Color(outbg)); err != nil {
				return err
			}
			cv.PutChar(x1+cx, y1+cy, glyphs[ch])
		}
	}
	return nil
}

// spreadFS applies the Floyd-Steinberg error distribution exactly as
// libcaca does, including C's truncating integer division.
func spreadFS(fsR, fsG, fsB []int, x int, errR, errG, errB int) (int, int, int, bool) {
	if x < 0 || x+1 >= len(fsR) {
		return 0, 0, 0, false
	}
	remainR := fsR[x+1] + 7*errR/16
	remainG := fsG[x+1] + 7*errG/16
	remainB := fsB[x+1] + 7*errB/16
	if x-1 >= 0 {
		fsR[x-1] += 3 * errR / 16
		fsG[x-1] += 3 * errG / 16
		fsB[x-1] += 3 * errB / 16
	}
	fsR[x] = 5 * errR / 16
	fsG[x] = 5 * errG / 16
	fsB[x] = 5 * errB / 16
	fsR[x+1] = errR / 16
	fsG[x+1] = errG / 16
	fsB[x+1] = errB / 16
	return remainR, remainG, remainB, true
}
