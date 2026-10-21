// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

// Command libcaca-pic renders an image file into the terminal.
//
// Usage:
//
//	go run ./cmd/libcaca-pic image.png
//	go run ./cmd/libcaca-pic -charset blocks -color full16 image.jpg
//
// Supported inputs are anything the Go standard library can decode plus
// JPEG, because image/jpeg is imported explicitly: PNG, JPEG and GIF.
//
// Press q or Ctrl-C to quit. When stdout is not a terminal the image is
// printed once as UTF-8 ANSI and the program exits.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/unboxlumen/libcaca"
)

func main() {
	var (
		charset    = flag.String("charset", "blocks", "character set: ascii, shades or blocks")
		colorMode  = flag.String("color", "full16", "colour mode: mono, gray, 8, 16, fullgray, full8 or full16")
		algorithm  = flag.String("dither", "fstein", "dithering: none, ordered2, ordered4, ordered8, random or fstein")
		width      = flag.Int("width", 0, "output width in cells (0 = terminal width)")
		height     = flag.Int("height", 0, "output height in cells (0 = keep aspect ratio)")
		brightness = flag.Float64("brightness", 1.0, "brightness multiplier")
		gamma      = flag.Float64("gamma", 1.0, "gamma correction")
		contrast   = flag.Float64("contrast", 1.0, "contrast multiplier")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] image.png|jpg|gif\n\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	img, dither, err := loadImage(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = img

	if err := dither.SetCharset(*charset); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := dither.SetColor(*colorMode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := dither.SetAlgorithm(*algorithm); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := dither.SetBrightness(*brightness); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := dither.SetGamma(*gamma); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := dither.SetContrast(*contrast); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	cols, rows := *width, *height
	if cols <= 0 {
		if w, _, ok := libcaca.TerminalSize(); ok {
			cols = w - 2
		} else {
			cols = 80
		}
	}
	if cols < 1 {
		cols = 1
	}
	// Terminal cells are roughly twice as tall as they are wide.
	if rows <= 0 {
		rows = cols * dither.Height() / dither.Width() / 2
		if rows < 1 {
			rows = 1
		}
	}
	if rows < 1 {
		rows = 1
	}

	c, err := libcaca.NewCanvas(cols, rows)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := dither.Draw(c, 0, 0, cols, rows); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	t, err := libcaca.NewTerminal(c)
	if err != nil {
		if !errors.Is(err, libcaca.ErrUnsupported) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		data, encErr := c.Encode("utf8")
		if encErr != nil {
			fmt.Fprintln(os.Stderr, encErr)
			os.Exit(1)
		}
		_, _ = os.Stdout.Write(data)
		return
	}
	defer t.Close()

	for {
		if err := t.Refresh(); err != nil {
			return
		}
		ev, ok := t.PollEvent(250 * time.Millisecond)
		if !ok {
			continue
		}
		switch ev.Type {
		case libcaca.EventKey:
			if ev.Key == 'q' || ev.Key == 'Q' || ev.Key == 3 {
				return
			}
		case libcaca.EventResize:
			if ev.Width > 0 && ev.Height > 0 {
				_ = c.Resize(ev.Width, ev.Height)
				if err := dither.Draw(c, 0, 0, c.Width(), c.Height()); err != nil {
					return
				}
			}
		case libcaca.EventQuit:
			return
		}
	}
}

func loadImage(path string) (image.Image, *libcaca.Dither, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	d, err := libcaca.NewDitherFromImage(img)
	if err != nil {
		return nil, nil, err
	}
	return img, d, nil
}
