// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestTerminalRefreshToFile(t *testing.T) {
	c := MustCanvas(4, 2)
	c.PutString(0, 0, "abc")
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()
	defer pw.Close()
	out, err := os.CreateTemp(t.TempDir(), "libcaca-out")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	term := &Terminal{canvas: c, in: pr, out: out}
	if err := term.Refresh(); err != nil {
		t.Fatal(err)
	}
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	first, err := io.ReadAll(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(first, []byte("abc")) {
		t.Fatalf("first refresh = %q", first)
	}

	if err := out.Truncate(0); err != nil {
		t.Fatal(err)
	}
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if err := term.Refresh(); err != nil {
		t.Fatal(err)
	}
	second, err := io.ReadAll(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("unchanged refresh wrote %q", second)
	}
}
