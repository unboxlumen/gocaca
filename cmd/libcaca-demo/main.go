// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

// Command libcaca-demo is a small interactive demo for the libcaca package.
//
// Run it from the module directory:
//
//	go run ./cmd/libcaca-demo
//
// Press q or Ctrl-C to quit. When stdout is not a terminal, the demo prints
// a single UTF-8 ANSI frame and exits.
package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/unboxlumen/libcaca"
)

func main() {
	c, err := libcaca.NewCanvas(60, 18)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// A terminal gives us animation and input. A pipe or file gets one frame.
	t, err := libcaca.NewTerminal(c)
	if err != nil {
		if !errors.Is(err, libcaca.ErrUnsupported) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		drawFrame(c, 28, 8, 0)
		data, encErr := c.Encode("utf8")
		if encErr != nil {
			fmt.Fprintln(os.Stderr, encErr)
			os.Exit(1)
		}
		_, _ = os.Stdout.Write(data)
		return
	}
	defer t.Close()

	bx, by := 10, 5
	dx, dy := 1, 1
	frame := 0
	for {
		w, h := c.Width(), c.Height()
		if w < 12 || h < 6 {
			if err := t.Refresh(); err != nil {
				return
			}
			ev, ok := t.PollEvent(100 * time.Millisecond)
			if !ok {
				continue
			}
			if ev.Type == libcaca.EventKey && (ev.Key == 'q' || ev.Key == 'Q' || ev.Key == 3) {
				return
			}
			if ev.Type == libcaca.EventResize && ev.Width > 0 && ev.Height > 0 {
				_ = c.Resize(ev.Width, ev.Height)
			}
			continue
		}
		bx += dx
		by += dy
		if bx <= 1 || bx >= w-3 {
			dx = -dx
			bx += 2 * dx
		}
		if by <= 2 || by >= h-3 {
			dy = -dy
			by += 2 * dy
		}
		drawFrame(c, bx, by, frame)
		if err := t.Refresh(); err != nil {
			return
		}
		frame++

		ev, ok := t.PollEvent(50 * time.Millisecond)
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
				if bx >= c.Width()-3 {
					bx = max(1, c.Width()-4)
					dx = -1
				}
				if by >= c.Height()-3 {
					by = max(2, c.Height()-4)
					dy = -1
				}
			}
		case libcaca.EventQuit:
			return
		}
	}
}

func drawFrame(c *libcaca.Canvas, bx, by, frame int) {
	w, h := c.Width(), c.Height()

	// Background and border.
	_ = c.SetColor(libcaca.LightGray, libcaca.Black)
	c.Clear()
	c.DrawThinBox(0, 0, w, h)

	// Title.
	_ = c.SetColor(libcaca.White, libcaca.Blue)
	c.FillBox(1, 0, max(0, w-2), 1, ' ')
	c.PutString(2, 0, " libcaca demo ")

	// A little colour strip.
	for i := 0; i < 8 && i < w-2; i++ {
		_ = c.SetColor(libcaca.Color(8+i), libcaca.Black)
		c.PutChar(i+1, 2, '█')
	}

	// The moving ball.
	_ = c.SetColor(libcaca.Yellow, libcaca.Black)
	c.FillEllipse(bx, by, 2, 2, '@')
	_ = c.SetColor(libcaca.Yellow, libcaca.Black)
	c.PutChar(bx, by, 'O')

	// Status line.
	_ = c.SetColor(libcaca.Green, libcaca.Black)
	c.PutString(2, h-3, fmt.Sprintf("frame %d · q/Ctrl-C to quit · resize me", frame))

	_ = c.SetColor(libcaca.DarkGray, libcaca.Black)
	c.PutString(2, h-2, "WTFPL v2 · pure Go · no cgo")
}
