// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import "errors"

// ErrUnsupported is returned when the terminal driver is not available.
var ErrUnsupported = errors.New("libcaca: terminal driver unsupported on this platform")

// EventType identifies a terminal event.
type EventType int

const (
	EventNone EventType = iota
	EventKey
	EventMouse
	EventResize
	EventQuit
)

// KeyCode identifies non-printable keys.
type KeyCode int

const (
	KeyNone KeyCode = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyInsert
	KeyDelete
	KeyBackspace
	KeyTab
	KeyEnter
	KeyEscape
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// Mod is a modifier bit mask.
type Mod int

const (
	ModShift Mod = 1 << iota
	ModAlt
	ModCtrl
)

// Event is a keyboard, mouse or resize event.
type Event struct {
	Type   EventType
	Key    rune
	Code   KeyCode
	Mod    Mod
	X, Y   int
	Button int
	Width  int
	Height int
}
