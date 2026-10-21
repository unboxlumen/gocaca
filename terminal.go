// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Terminal renders a canvas to an ANSI terminal and reads input events.
type Terminal struct {
	canvas    *Canvas
	in        *os.File
	out       *os.File
	state     *rawState
	prev      []string
	events    chan Event
	done      chan struct{}
	closeOnce sync.Once
}

// NewTerminal puts stdin into raw mode and enters the alternate screen.
func NewTerminal(c *Canvas) (*Terminal, error) {
	if c == nil {
		return nil, errors.New("libcaca: nil canvas")
	}
	in, out := os.Stdin, os.Stdout
	if !isTerminal(in.Fd()) {
		return nil, ErrUnsupported
	}
	state, err := makeRaw(in.Fd())
	if err != nil {
		return nil, err
	}
	t := &Terminal{
		canvas: c,
		in:     in,
		out:    out,
		state:  state,
		events: make(chan Event, 64),
		done:   make(chan struct{}),
	}
	if _, err := out.WriteString("\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H"); err != nil {
		restoreRaw(in.Fd(), state)
		return nil, err
	}
	go t.readLoop()
	startResizeNotify(t)
	return t, nil
}

// Canvas returns the canvas attached to the terminal.
func (t *Terminal) Canvas() *Canvas { return t.canvas }

// TerminalSize returns the size of the terminal attached to stdin.
func TerminalSize() (width, height int, ok bool) {
	w, h, err := getSize(os.Stdin.Fd())
	if err != nil || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}

// Size returns the terminal size, falling back to the canvas size.
func (t *Terminal) Size() (int, int) {
	if w, h, err := getSize(t.in.Fd()); err == nil && w > 0 && h > 0 {
		return w, h
	}
	return t.canvas.Width(), t.canvas.Height()
}

// Refresh draws changed rows. The first call draws the whole canvas.
func (t *Terminal) Refresh() error {
	cw, chh := t.canvas.Width(), t.canvas.Height()
	tw, th := t.Size()
	w, h := min(cw, tw), min(chh, th)
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	rows := make([]string, h)
	for y := 0; y < h; y++ {
		var b strings.Builder
		last := Attr(^uint32(0))
		have := false
		for x := 0; x < w; x++ {
			i := y*cw + x
			ch := t.canvas.cur.chars[i]
			if ch == Fullwidth {
				continue
			}
			a := t.canvas.cur.attrs[i]
			if !have || a != last {
				b.WriteString(sgr(a))
				last, have = a, true
			}
			b.WriteRune(ch)
		}
		b.WriteString("\x1b[0m")
		rows[y] = b.String()
	}
	if len(t.prev) != h {
		t.prev = make([]string, h)
		_, _ = t.out.WriteString("\x1b[2J")
	}
	var out strings.Builder
	for y := 0; y < h; y++ {
		if t.prev[y] == rows[y] {
			continue
		}
		out.WriteString("\x1b[")
		out.WriteString(strconv.Itoa(y + 1))
		out.WriteString(";1H")
		out.WriteString(rows[y])
		out.WriteString("\x1b[K")
		t.prev[y] = rows[y]
	}
	if out.Len() > 0 {
		_, err := t.out.WriteString(out.String())
		return err
	}
	return nil
}

// PollEvent waits for an event. A negative timeout blocks, zero polls, and
// a positive value waits at most that long.
func (t *Terminal) PollEvent(timeout time.Duration) (Event, bool) {
	if timeout < 0 {
		select {
		case ev := <-t.events:
			return ev, true
		case <-t.done:
			return Event{}, false
		}
	}
	if timeout == 0 {
		select {
		case ev := <-t.events:
			return ev, true
		case <-t.done:
			return Event{}, false
		default:
			return Event{}, false
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case ev := <-t.events:
		return ev, true
	case <-t.done:
		return Event{}, false
	case <-timer.C:
		return Event{}, false
	}
}

// Close restores the terminal. It is safe to call more than once.
func (t *Terminal) Close() error {
	var err error
	t.closeOnce.Do(func() {
		close(t.done)
		restoreRaw(t.in.Fd(), t.state)
		_, err = t.out.WriteString("\x1b[0m\x1b[?25h\x1b[?1049l")
	})
	return err
}

func (t *Terminal) readLoop() {
	buf := make([]byte, 256)
	var pending []byte
	for {
		select {
		case <-t.done:
			return
		default:
		}
		n, err := t.in.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			pending = parseEvents(pending, false, t.emit)
		}
		if err != nil {
			return
		}
		if n == 0 {
			pending = parseEvents(pending, true, t.emit)
		}
	}
}

func (t *Terminal) emit(ev Event) {
	select {
	case t.events <- ev:
	default:
	}
}

func emitKey(emit func(Event), code KeyCode, r rune, mod Mod) {
	emit(Event{Type: EventKey, Code: code, Key: r, Mod: mod})
}

func parseEvents(data []byte, flush bool, emit func(Event)) []byte {
	for len(data) > 0 {
		if data[0] == 0x1b {
			if len(data) == 1 {
				if flush {
					emitKey(emit, KeyEscape, 0, 0)
					return data[1:]
				}
				return data
			}
			switch data[1] {
			case '[':
				j := 2
				for j < len(data) && !(data[j] >= 0x40 && data[j] <= 0x7e) {
					j++
				}
				if j >= len(data) {
					if flush {
						emitKey(emit, KeyEscape, 0, 0)
						return nil
					}
					return data
				}
				handleCSI(string(data[2:j]), data[j], emit)
				data = data[j+1:]
				continue
			case 'O':
				if len(data) < 3 {
					if flush {
						emitKey(emit, KeyEscape, 0, 0)
						return nil
					}
					return data
				}
				handleSS3(data[2], emit)
				data = data[3:]
				continue
			default:
				emitKey(emit, KeyEscape, 0, 0)
				data = data[1:]
				continue
			}
		}

		b := data[0]
		switch b {
		case 0x03:
			emitKey(emit, KeyNone, 3, ModCtrl)
			data = data[1:]
			continue
		case '\r', '\n':
			emitKey(emit, KeyEnter, 0, 0)
			data = data[1:]
			continue
		case 0x7f, 0x08:
			emitKey(emit, KeyBackspace, 0, 0)
			data = data[1:]
			continue
		case '\t':
			emitKey(emit, KeyTab, 0, 0)
			data = data[1:]
			continue
		}
		if b < 0x20 {
			if b >= 0x01 && b <= 0x1a {
				emitKey(emit, KeyNone, rune('a'+b-1), ModCtrl)
			}
			data = data[1:]
			continue
		}
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			if !flush && !utf8.FullRune(data) {
				return data
			}
			emitKey(emit, KeyNone, r, 0)
			data = data[1:]
			continue
		}
		emitKey(emit, KeyNone, r, 0)
		data = data[size:]
	}
	return data
}

func handleCSI(seq string, final byte, emit func(Event)) {
	if final == 'M' || final == 'm' {
		if strings.HasPrefix(seq, "<") {
			handleMouse(seq[1:], final == 'm', emit)
			return
		}
	}
	if final == 'Z' && seq == "" {
		emitKey(emit, KeyTab, 0, ModShift)
		return
	}
	if final >= 'A' && final <= 'D' && seq != "" {
		// xterm modifier form: 1;5A etc.
		parts := strings.Split(seq, ";")
		if len(parts) == 2 {
			if n, err := strconv.Atoi(parts[1]); err == nil {
				mod := func(n int) Mod {
					var m Mod
					if n&1 != 0 {
						m |= ModShift
					}
					if n&2 != 0 {
						m |= ModAlt
					}
					if n&4 != 0 {
						m |= ModCtrl
					}
					return m
				}(n - 1)
				code := map[byte]KeyCode{'A': KeyUp, 'B': KeyDown, 'C': KeyRight, 'D': KeyLeft}[final]
				emitKey(emit, code, 0, mod)
				return
			}
		}
	}
	key := seq + string(final)
	switch key {
	case "A":
		emitKey(emit, KeyUp, 0, 0)
	case "B":
		emitKey(emit, KeyDown, 0, 0)
	case "C":
		emitKey(emit, KeyRight, 0, 0)
	case "D":
		emitKey(emit, KeyLeft, 0, 0)
	case "H", "1~", "7~":
		emitKey(emit, KeyHome, 0, 0)
	case "F", "4~", "8~":
		emitKey(emit, KeyEnd, 0, 0)
	case "2~":
		emitKey(emit, KeyInsert, 0, 0)
	case "3~":
		emitKey(emit, KeyDelete, 0, 0)
	case "5~":
		emitKey(emit, KeyPageUp, 0, 0)
	case "6~":
		emitKey(emit, KeyPageDown, 0, 0)
	case "11~":
		emitKey(emit, KeyF1, 0, 0)
	case "12~":
		emitKey(emit, KeyF2, 0, 0)
	case "13~":
		emitKey(emit, KeyF3, 0, 0)
	case "14~":
		emitKey(emit, KeyF4, 0, 0)
	case "15~":
		emitKey(emit, KeyF5, 0, 0)
	case "17~":
		emitKey(emit, KeyF6, 0, 0)
	case "18~":
		emitKey(emit, KeyF7, 0, 0)
	case "19~":
		emitKey(emit, KeyF8, 0, 0)
	case "20~":
		emitKey(emit, KeyF9, 0, 0)
	case "21~":
		emitKey(emit, KeyF10, 0, 0)
	case "23~":
		emitKey(emit, KeyF11, 0, 0)
	case "24~":
		emitKey(emit, KeyF12, 0, 0)
	}
}

func handleSS3(final byte, emit func(Event)) {
	switch final {
	case 'P':
		emitKey(emit, KeyF1, 0, 0)
	case 'Q':
		emitKey(emit, KeyF2, 0, 0)
	case 'R':
		emitKey(emit, KeyF3, 0, 0)
	case 'S':
		emitKey(emit, KeyF4, 0, 0)
	case 'A':
		emitKey(emit, KeyUp, 0, 0)
	case 'B':
		emitKey(emit, KeyDown, 0, 0)
	case 'C':
		emitKey(emit, KeyRight, 0, 0)
	case 'D':
		emitKey(emit, KeyLeft, 0, 0)
	case 'H':
		emitKey(emit, KeyHome, 0, 0)
	case 'F':
		emitKey(emit, KeyEnd, 0, 0)
	}
}

func handleMouse(seq string, release bool, emit func(Event)) {
	parts := strings.Split(seq, ";")
	if len(parts) != 3 {
		return
	}
	b, e1 := strconv.Atoi(parts[0])
	x, e2 := strconv.Atoi(parts[1])
	y, e3 := strconv.Atoi(parts[2])
	if e1 != nil || e2 != nil || e3 != nil {
		return
	}
	var mod Mod
	if b&4 != 0 {
		mod |= ModShift
	}
	if b&8 != 0 {
		mod |= ModAlt
	}
	if b&16 != 0 {
		mod |= ModCtrl
	}
	button := b & 3
	if release {
		button = -1
	}
	emit(Event{Type: EventMouse, X: x - 1, Y: y - 1, Button: button, Mod: mod})
}

// startResizeNotify is implemented per platform.
func startResizeNotify(t *Terminal) {
	go func() {
		ch, stop := resizeChannel()
		defer stop()
		if ch == nil {
			return
		}
		for {
			select {
			case <-t.done:
				return
			case <-ch:
				w, h := t.Size()
				t.emit(Event{Type: EventResize, Width: w, Height: h})
			}
		}
	}()
}
