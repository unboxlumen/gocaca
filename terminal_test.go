// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"strings"
	"testing"
)

func collectEvents(t *testing.T, data []byte, flush bool) []Event {
	t.Helper()
	var events []Event
	rest := parseEvents(data, flush, func(e Event) { events = append(events, e) })
	if len(rest) != 0 {
		t.Fatalf("unparsed remainder: %q", rest)
	}
	return events
}

func TestParseArrowKeys(t *testing.T) {
	for seq, want := range map[string]KeyCode{
		"\x1b[A":   KeyUp,
		"\x1b[B":   KeyDown,
		"\x1b[C":   KeyRight,
		"\x1b[D":   KeyLeft,
		"\x1b[H":   KeyHome,
		"\x1b[F":   KeyEnd,
		"\x1b[2~":  KeyInsert,
		"\x1b[3~":  KeyDelete,
		"\x1b[5~":  KeyPageUp,
		"\x1b[6~":  KeyPageDown,
		"\x1b[15~": KeyF5,
		"\x1b[24~": KeyF12,
	} {
		evs := collectEvents(t, []byte(seq), false)
		if len(evs) != 1 || evs[0].Code != want {
			t.Fatalf("%q = %#v, want code %v", seq, evs, want)
		}
	}
}

func TestParsePlainKeys(t *testing.T) {
	evs := collectEvents(t, []byte("a\r\x7f\t\x03"), false)
	if len(evs) != 5 {
		t.Fatalf("events = %#v", evs)
	}
	if evs[0].Key != 'a' || evs[0].Code != KeyNone {
		t.Fatalf("printable key = %#v", evs[0])
	}
	if evs[1].Code != KeyEnter {
		t.Fatalf("enter = %#v", evs[1])
	}
	if evs[2].Code != KeyBackspace {
		t.Fatalf("backspace = %#v", evs[2])
	}
	if evs[3].Code != KeyTab {
		t.Fatalf("tab = %#v", evs[3])
	}
	if evs[4].Key != 3 || evs[4].Mod != ModCtrl {
		t.Fatalf("ctrl-c = %#v", evs[4])
	}
}

func TestParseUTF8(t *testing.T) {
	evs := collectEvents(t, []byte("é"), false)
	if len(evs) != 1 || evs[0].Key != 'é' {
		t.Fatalf("events = %#v", evs)
	}
}

func TestParseIncomplete(t *testing.T) {
	data := []byte("\x1b[")
	if rest := parseEvents(data, false, func(Event) {}); string(rest) != string(data) {
		t.Fatalf("incomplete sequence was consumed: %q", rest)
	}
	evs := collectEvents(t, data, true)
	if len(evs) != 1 || evs[0].Code != KeyEscape {
		t.Fatalf("flush = %#v", evs)
	}
}

func TestParseMouse(t *testing.T) {
	evs := collectEvents(t, []byte("\x1b[<0;10;5M"), false)
	if len(evs) != 1 {
		t.Fatalf("events = %#v", evs)
	}
	ev := evs[0]
	if ev.Type != EventMouse || ev.X != 9 || ev.Y != 4 || ev.Button != 0 {
		t.Fatalf("mouse = %#v", ev)
	}
	evs = collectEvents(t, []byte("\x1b[<0;10;5m"), false)
	if len(evs) != 1 || evs[0].Button != -1 {
		t.Fatalf("mouse release = %#v", evs)
	}
}

func TestSGR(t *testing.T) {
	s := sgr(ANSIAttr(Red, Blue, Bold))
	if !strings.Contains(s, "1;") || !strings.Contains(s, "31") || !strings.Contains(s, "44") {
		t.Fatalf("sgr = %q", s)
	}
}
