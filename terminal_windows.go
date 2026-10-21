//go:build windows

// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import "os"

type rawState struct{}

func isTerminal(uintptr) bool { return false }

func makeRaw(uintptr) (*rawState, error) { return nil, ErrUnsupported }

func restoreRaw(uintptr, *rawState) {}

func getSize(uintptr) (int, int, error) { return 0, 0, ErrUnsupported }

func resizeChannel() (<-chan os.Signal, func()) { return nil, func() {} }
