// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

type cacaFrameInfo struct {
	width, height uint32
	duration      uint32
	attr          uint32
	x, y          int32
	handlex       int32
	handley       int32
}

// encodeCaca writes the native big-endian canvas format.
func (c *Canvas) encodeCaca() ([]byte, error) {
	frames := len(c.frames)
	if uint64(frames) > (1<<32 - 1) {
		return nil, errors.New("libcaca: too many frames")
	}
	controlSize := uint64(16 + 32*frames)
	var dataSize uint64
	for _, f := range c.frames {
		dataSize += uint64(f.width) * uint64(f.height) * 8
	}
	if controlSize > 1<<32-1 || dataSize > 1<<32-1 {
		return nil, errors.New("libcaca: canvas too large for native format")
	}

	var b bytes.Buffer
	b.Write([]byte{0xca, 0xca, 'C', 'V'})
	writeU32(&b, uint32(controlSize))
	writeU32(&b, uint32(dataSize))
	writeU16(&b, 1)
	writeU32(&b, uint32(frames))
	writeU16(&b, 0)

	for _, f := range c.frames {
		writeU32(&b, uint32(f.width))
		writeU32(&b, uint32(f.height))
		writeU32(&b, 0)
		writeU32(&b, uint32(f.curattr))
		writeU32(&b, uint32(int32(f.x)))
		writeU32(&b, uint32(int32(f.y)))
		writeU32(&b, uint32(int32(f.handlex)))
		writeU32(&b, uint32(int32(f.handley)))
	}
	for _, f := range c.frames {
		for i := range f.chars {
			writeU32(&b, uint32(f.chars[i]))
			writeU32(&b, uint32(f.attrs[i]))
		}
	}
	return b.Bytes(), nil
}

func decodeCaca(data []byte) (*Canvas, error) {
	if len(data) < 20 {
		return nil, errors.New("libcaca: truncated native header")
	}
	if data[0] != 0xca || data[1] != 0xca || data[2] != 'C' || data[3] != 'V' {
		return nil, errors.New("libcaca: bad native magic")
	}
	controlSize := binary.BigEndian.Uint32(data[4:])
	dataSize := binary.BigEndian.Uint32(data[8:])
	version := binary.BigEndian.Uint16(data[12:])
	frames := binary.BigEndian.Uint32(data[14:])
	_ = binary.BigEndian.Uint16(data[18:])
	if version != 1 {
		return nil, fmt.Errorf("libcaca: unsupported native version %d", version)
	}
	if frames == 0 {
		return nil, errors.New("libcaca: native file has no frames")
	}
	if uint64(controlSize) < 16+32*uint64(frames) {
		return nil, errors.New("libcaca: invalid control size")
	}
	if uint64(len(data)) < 4+uint64(controlSize)+uint64(dataSize) {
		return nil, errors.New("libcaca: truncated native data")
	}

	infos := make([]cacaFrameInfo, frames)
	var expected uint64
	var xmin, ymin, xmax, ymax int32
	for i := uint32(0); i < frames; i++ {
		off := 4 + 16 + int(i)*32
		fi := cacaFrameInfo{
			width:    binary.BigEndian.Uint32(data[off:]),
			height:   binary.BigEndian.Uint32(data[off+4:]),
			duration: binary.BigEndian.Uint32(data[off+8:]),
			attr:     binary.BigEndian.Uint32(data[off+12:]),
			x:        int32(binary.BigEndian.Uint32(data[off+16:])),
			y:        int32(binary.BigEndian.Uint32(data[off+20:])),
			handlex:  int32(binary.BigEndian.Uint32(data[off+24:])),
			handley:  int32(binary.BigEndian.Uint32(data[off+28:])),
		}
		if fi.width != 0 && uint64(fi.height) > (^uint64(0)/8)/uint64(fi.width) {
			return nil, errors.New("libcaca: native frame size overflow")
		}
		expected += uint64(fi.width) * uint64(fi.height) * 8
		infos[i] = fi
		if -fi.handlex < xmin {
			xmin = -fi.handlex
		}
		if -fi.handley < ymin {
			ymin = -fi.handley
		}
		if int32(fi.width)-fi.handlex > xmax {
			xmax = int32(fi.width) - fi.handlex
		}
		if int32(fi.height)-fi.handley > ymax {
			ymax = int32(fi.height) - fi.handley
		}
	}
	if expected != uint64(dataSize) {
		return nil, errors.New("libcaca: native data size mismatch")
	}
	cw, ch := int(xmax-xmin), int(ymax-ymin)
	if cw < 0 || ch < 0 {
		return nil, errors.New("libcaca: invalid native canvas extent")
	}

	nc := &Canvas{}
	offset := 4 + int(controlSize)
	for i, fi := range infos {
		f := newFrame(cw, ch, Attr(fi.attr), fmt.Sprintf("frame#%08x", i))
		for j := range f.attrs {
			f.attrs[j] = Attr(fi.attr)
		}
		for n := 0; n < int(fi.width)*int(fi.height); n++ {
			x := (n % int(fi.width)) - int(fi.handlex) - int(xmin)
			y := (n / int(fi.width)) - int(fi.handley) - int(ymin)
			chv := rune(binary.BigEndian.Uint32(data[offset+8*n:]))
			at := Attr(binary.BigEndian.Uint32(data[offset+8*n+4:]))
			if x >= 0 && x < cw && y >= 0 && y < ch {
				f.chars[y*cw+x] = chv
				f.attrs[y*cw+x] = at
			}
		}
		offset += int(fi.width) * int(fi.height) * 8
		f.x = int(fi.x - fi.handlex)
		f.y = int(fi.y - fi.handley)
		f.handlex = int(-xmin)
		f.handley = int(-ymin)
		nc.frames = append(nc.frames, f)
	}
	nc.cur = nc.frames[0]
	nc.autoinc = len(nc.frames)
	return nc, nil
}

func writeU16(b *bytes.Buffer, v uint16) {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], v)
	b.Write(buf[:])
}

func writeU32(b *bytes.Buffer, v uint32) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	b.Write(buf[:])
}
