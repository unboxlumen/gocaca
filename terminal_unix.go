//go:build linux || darwin

// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

type rawState struct {
	termios syscall.Termios
}

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) syscall.Errno {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, req, uintptr(arg), 0, 0, 0)
	return errno
}

func isTerminal(fd uintptr) bool {
	var t syscall.Termios
	return ioctl(fd, ioctlGetTermios, unsafe.Pointer(&t)) == 0
}

func makeRaw(fd uintptr) (*rawState, error) {
	var t syscall.Termios
	if errno := ioctl(fd, ioctlGetTermios, unsafe.Pointer(&t)); errno != 0 {
		return nil, errno
	}
	old := t
	t.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK |
		syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	t.Oflag &^= syscall.OPOST
	t.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	t.Cflag &^= syscall.CSIZE | syscall.PARENB
	t.Cflag |= syscall.CS8 | syscall.CREAD | syscall.CLOCAL
	// Block until at least one byte is available. VMIN=0/VTIME=1 makes
	// read(2) return zero bytes, which os.File.Read reports as io.EOF.
	t.Cc[syscall.VMIN] = 1
	t.Cc[syscall.VTIME] = 0
	if errno := ioctl(fd, ioctlSetTermios, unsafe.Pointer(&t)); errno != 0 {
		return nil, errno
	}
	return &rawState{termios: old}, nil
}

func restoreRaw(fd uintptr, st *rawState) {
	if st == nil {
		return
	}
	_ = ioctl(fd, ioctlSetTermios, unsafe.Pointer(&st.termios))
}

type winsize struct {
	Row, Col, Xpixel, Ypixel uint16
}

func getSize(fd uintptr) (int, int, error) {
	var ws winsize
	if errno := ioctl(fd, ioctlGetWinsize, unsafe.Pointer(&ws)); errno != 0 {
		return 0, 0, errno
	}
	return int(ws.Col), int(ws.Row), nil
}

func resizeChannel() (<-chan os.Signal, func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	return ch, func() { signal.Stop(ch) }
}
