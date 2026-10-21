// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"fmt"
	"strings"
)

// Formats returns the supported codec names.
func Formats() []string { return []string{"text", "ansi", "utf8", "caca"} }

// Encode serialises the canvas in the requested format.
func (c *Canvas) Encode(format string) ([]byte, error) {
	switch strings.ToLower(format) {
	case "text":
		return c.encodeText(), nil
	case "ansi":
		return c.encodeANSI(false), nil
	case "utf8":
		return c.encodeANSI(true), nil
	case "caca":
		return c.encodeCaca()
	default:
		return nil, fmt.Errorf("libcaca: unknown format %q", format)
	}
}

// Decode replaces the canvas contents with data in the requested format.
// An empty format autodetects the native, ANSI and text formats.
func (c *Canvas) Decode(data []byte, format string) error {
	f := strings.ToLower(format)
	if f == "" {
		switch {
		case len(data) >= 4 && data[0] == 0xca && data[1] == 0xca && data[2] == 'C' && data[3] == 'V':
			f = "caca"
		case containsANSI(data):
			f = "ansi"
		default:
			f = "text"
		}
	}
	var (
		nc  *Canvas
		err error
	)
	switch f {
	case "text":
		nc, err = decodeText(data)
	case "ansi":
		nc, err = decodeANSI(data, false)
	case "utf8":
		nc, err = decodeANSI(data, true)
	case "caca":
		nc, err = decodeCaca(data)
	default:
		return fmt.Errorf("libcaca: unknown format %q", format)
	}
	if err != nil {
		return err
	}
	c.replaceWith(nc)
	return nil
}

func (c *Canvas) replaceWith(nc *Canvas) {
	c.frames = nc.frames
	c.frame = nc.frame
	c.cur = c.frames[c.frame]
	c.autoinc = nc.autoinc
	c.dirty = c.dirty[:0]
	if !c.dirtyDisabled() {
		c.addDirtyRect(0, 0, c.cur.width, c.cur.height)
	}
}

func containsANSI(data []byte) bool {
	for i := 0; i+1 < len(data); i++ {
		if data[i] == 0x1b && data[i+1] == '[' {
			return true
		}
	}
	return false
}
