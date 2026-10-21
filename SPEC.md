# libcaca Go port specification

This document describes the Go API and the behaviour of the port. The
implementation is a direct Go port of the WTFPL-licensed core of libcaca.
GPL and LGPL files from the parent repository are explicitly out of scope;
see `PROVENANCE.md`.

Where this document and the Go code differ, the code is authoritative. The API
was aligned with the C semantics during porting. Notable differences from
early drafts of this document:

- `NewFrame` returns the inserted index and does not switch the active frame.
- `SetBoundaries` preserves the cursor and handle relative to the cropped
  content.
- `FillTriangle` uses a scanline rasteriser that includes boundary cells.
- `PutString` handles newlines, carriage returns and tabs because that is more
  useful in Go; the C function does not.

## 1. Scope

Package `libcaca` is a pure Go colour ASCII-art canvas library. It must compile
with `go build ./...` and pass `go test ./...` with no third-party modules and
no cgo.

In scope:

- canvas with per-cell character and attribute storage
- multiple frames per canvas
- dirty-rectangle tracking
- ANSI / ARGB4444 attributes and styles
- UTF-8 / UTF-32 / CP437 / ASCII helpers
- line, box, circle, ellipse, triangle drawing
- invert, flip, flop, rotation and stretch transforms
- image dithering
- text, ANSI, UTF-8 and native binary codecs
- ANSI terminal output and keyboard/mouse event input on Linux and macOS
- examples and tests

Out of scope for this MVP:

- bitmap fonts and FIGfont
- HTML, SVG, TGA, PostScript, troff, IRC and BBcode exports
- X11, OpenGL, Cocoa, Win32, DOS conio and VGA drivers
- C ABI compatibility, C headers, `libcaca.so.0`
- third-party bindings

## 2. Licence

gocaca is released under the WTFPL v2. It is a Go port of the **core** of
libcaca, the `caca/` library sources at
<https://github.com/cacalabs/libcaca>; that core is itself distributed under
the WTFPL v2 (upstream `COPYING`). Only that core is used as the reference —
the GPL/LGPL files and the unlicensed font data elsewhere in the upstream
tree are out of scope. All files are WTFPL v2. Put this header at the top of
every `.go` file:

```go
// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.
```

The implementation is a direct port of the WTFPL-licensed libcaca core, so
copying from those files is permitted. Do not include the GPL/LGPL files or the generated font data listed in
`PROVENANCE.md`. Keep the CP437 and
fullwidth mappings documented as factual public mappings.

## 3. Package layout

```text
libcaca/
  LICENSE
  go.mod
  README.md
  SPEC.md
  attr.go
  attr_test.go
  charset.go
  charset_test.go
  canvas.go
  canvas_test.go
  draw.go
  draw_test.go
  transform.go
  transform_test.go
  codec.go
  codec_text.go
  codec_ansi.go
  codec_caca.go
  codec_test.go
  event.go
  terminal.go
  terminal_unix.go
  terminal_windows.go
  internal_test.go
```

Additional files are allowed if the package stays cohesive.

## 4. Attributes

```go
type Color uint8

const (
    Black Color = iota
    Blue
    Green
    Cyan
    Red
    Magenta
    Brown
    LightGray
    DarkGray
    LightBlue
    LightGreen
    LightCyan
    LightRed
    LightMagenta
    Yellow
    White
    Default     // terminal default colour
    Transparent // let the background show through
)

type Style uint8

const (
    Bold Style = 1 << iota
    Italics
    Underline
    Blink
)

type Attr uint32
```

`Attr` layout, bit-compatible with libcaca:

- bits 0..3: `Style`
- bits 4..17: foreground, 14 bits
- bits 18..31: background, 14 bits
- an ANSI component is stored as `index|0x40` (below `0x50`); `0x50` and
  `0x60` mean Default and Transparent
- any other value is a 14-bit ARGB4444 value
- `Attr.ANSI`, `Attr.FG`, `Attr.BG`, `Attr.RGB12FG`, `Attr.RGB12BG` and
  `Attr.ARGB64` mirror the corresponding libcaca accessors

Constructor and accessors:

```go
func ANSIAttr(fg, bg Color, style Style) Attr
func RGBAttr(fg, bg color.RGBA, style Style) Attr

func (a Attr) FG() Color          // Default when the foreground is RGB
func (a Attr) BG() Color          // Default when the background is RGB
func (a Attr) Style() Style
func (a Attr) IsRGB() bool
func (a Attr) RGB() (fg, bg color.RGBA, ok bool)
func (a Attr) WithStyle(s Style) Attr
```

`RGBAttr` quantises each channel to 4 bits by `v >> 4`. `RGB` expands back with
`v<<4 | v`, so the result is the quantised value. `ANSIAttr` clamps an
out-of-range colour to `LightGray` for the foreground and `Black` for the
background. A zero `Attr` is valid and means `ANSIAttr(Black, Black, 0)`; the
canvas default is `ANSIAttr(LightGray, Black, 0)`.

## 5. Characters

```go
const Fullwidth rune = 0x000FFFFE

func UTF8ToUTF32(s string) []rune
func UTF32ToUTF8(r []rune) string
func UTF32ToCP437(r rune) byte
func CP437ToUTF32(b byte) rune
func UTF32ToASCII(r rune) byte
func IsFullwidth(r rune) bool
```

Rules:

- `UTF8ToUTF32` decodes UTF-8. An invalid byte becomes `utf8.RuneError` and
  consumes one byte, matching `unicode/utf8.DecodeRuneInString`.
- `UTF32ToUTF8` encodes each rune; invalid runes become `utf8.RuneError`.
- `UTF32ToCP437` returns `'?'` for runes outside CP437.
- `CP437ToUTF32` maps bytes 0x00..0x7f to themselves and 0x80..0xff through
  the standard CP437 table.
- `UTF32ToASCII` keeps printable ASCII (0x20..0x7e), maps common Latin-1
  letters and accents to their closest ASCII letter, maps tab/newline/carriage
  return through, and returns `'?'` otherwise.
- `IsFullwidth` returns true for East Asian Wide/Fullwidth ranges. At minimum
  include: U+1100..U+115F, U+2E80..U+303E, U+3041..U+33FF, U+3400..U+4DBF,
  U+4E00..U+9FFF, U+A000..U+A4CF, U+AC00..U+D7A3, U+F900..U+FAFF,
  U+FE10..U+FE19, U+FE30..U+FE6F, U+FF00..U+FF60, U+FFE0..U+FFE6,
  U+1F300..U+1F64F, U+1F900..U+1F9FF, U+20000..U+2FFFD, U+30000..U+3FFFD.

## 6. Canvas

```go
type Point struct{ X, Y int }

type Canvas struct { /* private */ }

func NewCanvas(width, height int) (*Canvas, error)
func MustCanvas(width, height int) *Canvas

func (c *Canvas) Width() int
func (c *Canvas) Height() int
func (c *Canvas) Resize(width, height int) error

func (c *Canvas) Clear()
func (c *Canvas) PutChar(x, y int, ch rune) bool
func (c *Canvas) Char(x, y int) rune
func (c *Canvas) PutAttr(x, y int, a Attr) bool
func (c *Canvas) Attr(x, y int) Attr
func (c *Canvas) SetAttr(a Attr)
func (c *Canvas) CurrentAttr() Attr
func (c *Canvas) SetColor(fg, bg Color) error
func (c *Canvas) SetRGB(fg, bg color.RGBA)
func (c *Canvas) PutString(x, y int, s string) int
func (c *Canvas) Printf(x, y int, format string, args ...any) int

func (c *Canvas) SetHandle(x, y int)
func (c *Canvas) Handle() (x, y int)
func (c *Canvas) Blit(x, y int, src *Canvas, mask *Canvas) error
func (c *Canvas) SetBoundaries(x, y, width, height int) error

func (c *Canvas) FrameCount() int
func (c *Canvas) CurrentFrame() int
func (c *Canvas) SetFrame(i int) error
func (c *Canvas) NewFrame(i int) error
func (c *Canvas) FreeFrame(i int) error
func (c *Canvas) FrameName(i int) string
func (c *Canvas) SetFrameName(i int, name string) error

func (c *Canvas) DirtyRectCount() int
func (c *Canvas) DirtyRect(i int) (x, y, width, height int, ok bool)
func (c *Canvas) AddDirtyRect(x, y, width, height int)
func (c *Canvas) ClearDirtyRects()
func (c *Canvas) EnableDirtyRects()
func (c *Canvas) DisableDirtyRects()
```

Semantics:

- `NewCanvas` accepts zero width and/or height. Negative dimensions, or
  dimensions whose product overflows `int`, return an error. A nil canvas is
  never returned with a nil error.
- `MustCanvas` panics on error.
- `Resize` preserves the overlapping top-left region, fills new cells with
  `' '` and the current attribute, and clips content. Negative or overflowing
  sizes return an error and leave the canvas unchanged.
- `Clear` fills every cell with `' '` and the current attribute, and clears
  dirty rectangles.
- `PutChar` clips out-of-bounds coordinates and returns false. `ch == 0` is
  stored as `' '`.
  - If `IsFullwidth(ch)` and `x+1 < width`, store `ch` at `(x,y)` and
    `Fullwidth` at `(x+1,y)`.
  - If `IsFullwidth(ch)` and `x+1 >= width`, store `' '` and return false.
  - If the cell being overwritten is `Fullwidth`, also clear the preceding
    cell to `' '` when it is a fullwidth glyph.
  - If the cell at `x+1` is `Fullwidth`, clear it to `' '` when overwriting
    the previous half of a fullwidth pair.
- `Char` returns `' '` for out-of-bounds coordinates. It returns `Fullwidth`
  for the trailing half of a fullwidth glyph.
- `PutAttr` clips and returns false when out of bounds. It never changes the
  character.
- `SetAttr` sets the current attribute. `SetColor` validates each colour:
  valid values are 0..16 and 32. Invalid values return an error and leave the
  current attribute unchanged. `Transparent` is accepted for both foreground
  and background in this MVP.
- `SetRGB` sets the current attribute to `RGBAttr(fg, bg, 0)`.
- `PutString` decodes UTF-8 and writes cells from `(x,y)`. A newline moves to
  `(0, y+1)`; a carriage return sets `x=0`; a tab advances to the next
  multiple of 8. It returns the number of cells written, excluding skipped
  trailing fullwidth cells. Invalid UTF-8 is written as `utf8.RuneError`.
- `Printf` formats with `fmt.Sprintf` and calls `PutString`.
- `SetHandle` stores an offset used by `Blit` and by terminal cursor
  positioning. The default handle is `(0,0)`.
- `Blit` copies the source canvas onto the destination at `(x,y)`, clipped to
  the destination. Source cells outside the source bounds are ignored. If
  `mask != nil`, a mask cell containing `' '` or `Fullwidth` is skipped; other
  mask cells copy the source cell and attribute. The source is not modified.
  A nil source returns an error.
- `SetBoundaries(x,y,w,h)` changes the visible canvas to the rectangle
  `(x,y,w,h)` of the current frame. The new canvas has size `w x h`; content
  is shifted by `(-x,-y)`; cells outside the old canvas become `' '` with the
  current attribute. Negative or overflowing sizes return an error. The
  handle becomes `(-x,-y)`.
- Frames: every canvas starts with one frame. `NewFrame(i)` inserts a new
  empty frame at index `i` (`0..FrameCount()`), makes it current, and returns
  an error for an invalid index. `FreeFrame(i)` removes a non-current frame;
  freeing the last remaining frame returns an error. `SetFrame` saves the
  current frame's geometry and makes frame `i` current. `SetFrameName` returns
  an error for an invalid index. `FrameName` returns `""` for an invalid index.
- Each frame stores its own width, height, chars, attrs, handle and name.
  Switching frames restores all of them.
- Dirty rectangles are clipped to the canvas. If dirty tracking is disabled,
  `AddDirtyRect` is a no-op. `PutChar`, `PutAttr`, `PutString`, drawing and
  transforms add dirty rectangles automatically when tracking is enabled.
  `DirtyRect(i)` returns `ok == false` for an invalid index. The list is
  cleared by `ClearDirtyRects` and by `Clear`.
- A `Canvas` is not safe for concurrent use by multiple goroutines.

## 7. Drawing

```go
func (c *Canvas) DrawLine(x1, y1, x2, y2 int, ch rune)
func (c *Canvas) DrawPolyline(points []Point, ch rune)
func (c *Canvas) DrawThinLine(x1, y1, x2, y2 int)
func (c *Canvas) DrawThinPolyline(points []Point)
func (c *Canvas) DrawBox(x, y, width, height int, ch rune)
func (c *Canvas) FillBox(x, y, width, height int, ch rune)
func (c *Canvas) DrawThinBox(x, y, width, height int)
func (c *Canvas) DrawCircle(cx, cy, radius int, ch rune)
func (c *Canvas) DrawEllipse(cx, cy, rx, ry int, ch rune)
func (c *Canvas) FillEllipse(cx, cy, rx, ry int, ch rune)
func (c *Canvas) DrawTriangle(x1, y1, x2, y2, x3, y3 int, ch rune)
func (c *Canvas) FillTriangle(x1, y1, x2, y2, x3, y3 int, ch rune)
```

Rules:

- All drawing uses the current attribute and clips to the canvas.
- `DrawLine` uses integer Bresenham and includes both endpoints.
- `DrawPolyline` draws consecutive segments; a single point draws one cell.
- `DrawThinLine` uses Unicode box-drawing characters: `─`, `│`, `╱`, `╲`,
  plus `┼` for crossings. `DrawThinBox` uses `┌`, `┐`, `└`, `┘`, `─`, `│`.
- `FillBox` fills every cell in the rectangle. A zero or negative width or
  height does nothing.
- `DrawCircle`/`DrawEllipse` use midpoint algorithms and include the outline.
  `FillEllipse` fills the interior.
- Triangle rasterisation must fill every pixel whose centre lies inside the
  triangle; use an integer edge-function rasteriser. Degenerate triangles
  draw nothing. `DrawTriangle` draws the three edges.

## 8. Transforms

```go
func (c *Canvas) Invert()
func (c *Canvas) Flip()
func (c *Canvas) Flop()
func (c *Canvas) Rotate180()
func (c *Canvas) RotateLeft()
func (c *Canvas) RotateRight()
func (c *Canvas) StretchHorizontal()
func (c *Canvas) StretchVertical()
```

Rules:

- `Invert` swaps foreground and background colours on every cell, preserving
  styles. RGB channels are inverted (`255-v`) after expansion.
- `Flip` reverses rows (top to bottom). `Flop` reverses columns (left to
  right). Both are involutions.
- `Rotate180` is `Flip` + `Flop`. `RotateLeft` rotates 90 degrees
  counter-clockwise; `RotateRight` rotates 90 degrees clockwise. Applying any
  rotation four times restores the original canvas.
- `StretchHorizontal` doubles the width by duplicating each column.
  `StretchVertical` doubles the height by duplicating each row. Attributes
  are duplicated with the cells.
- Transforms preserve the handle, frame count and frame names. They operate
  on the current frame and resize only that frame.

## 9. Codecs

```go
func (c *Canvas) Encode(format string) ([]byte, error)
func (c *Canvas) Decode(data []byte, format string) error
func Formats() []string
```

`Formats()` returns `[]string{"text", "ansi", "utf8", "caca"}`.

### text

Encode every cell row by row, skipping `Fullwidth` trailing cells, and append
`\n` after each row. Preserve trailing spaces. Decode as UTF-8 text: clear the
canvas, resize it to the longest line and the number of lines, and write the
characters with the current attribute. Tabs advance to the next multiple of
8; `\r` is ignored; a missing trailing newline is allowed.

### ansi

Encode using ANSI SGR sequences:

- reset with `\x1b[0m` at the start and end of the document
- styles: bold `1`, italics `3`, underline `4`, blink `5`
- ANSI foreground 0..7: `30+n`; 8..15: `90+n-8`
- ANSI background 0..7: `40+n`; 8..15: `100+n-8`
- `Default` foreground: `39`; `Default` background: `49`
- `Transparent` background: `49`
- RGB foreground: `38;2;r;g;b`; RGB background: `48;2;r;g;b`
- emit a new SGR sequence only when the attribute changes
- append `\n` after each row, including the last row

Decode both ANSI and UTF-8 variants through the same parser. For `ansi`,
bytes 0x80..0xff in text positions are decoded as CP437; for `utf8`, they are
decoded as UTF-8. Recognise `\x1b[...m` SGR sequences, `\r`, `\n`, and printable
characters; ignore other CSI sequences. Reset clears the current attribute to
the canvas default. Resize the canvas to the longest line and the number of
lines.

### caca

Native binary format, big-endian:

```text
magic            4 bytes   0xCA 0xCA 'C' 'V'
control_size     uint32    bytes of control data
data_size        uint32    bytes of cell data
version          uint16    1
frames           uint32
flags            uint16    0
frame_info[frames]:
  width          uint32
  height         uint32
  duration       uint32    (0)
  attr           uint32
  x              int32
  y              int32
  handlex        int32
  handley        int32
cell_data[frames]:
  char           uint32
  attr           uint32
```

`control_size = 16 + 32*frames`; `data_size = sum(width*height*8)`. Decode
validates the magic, sizes and frame count; on error it returns an error and
leaves the canvas unchanged. Encode writes all frames. The native codec is
for interoperability with `.caca` files; do not copy the C implementation.

## 10. Events and terminal

```go
type EventType int

const (
    EventNone EventType = iota
    EventKey
    EventMouse
    EventResize
    EventQuit
)

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

type Mod int

const (
    ModShift Mod = 1 << iota
    ModAlt
    ModCtrl
)

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

type Terminal struct { /* private */ }

func NewTerminal(c *Canvas) (*Terminal, error)
func (t *Terminal) Refresh() error
func (t *Terminal) Close() error
func (t *Terminal) Size() (width, height int)
func (t *Terminal) PollEvent(timeout time.Duration) (Event, bool)
func (t *Terminal) Canvas() *Canvas
```

Terminal semantics:

- `NewTerminal` requires a non-nil canvas and a terminal on stdin/stdout.
  On unsupported platforms it returns `ErrUnsupported`.
- On Unix (Linux and macOS), switch stdin to raw mode, hide the cursor, enter
  the alternate screen, and arrange for `Close` to restore the previous
  termios state, show the cursor and leave the alternate screen. `Close` is
  idempotent.
- `Refresh` renders the canvas using ANSI sequences. It should avoid a full
  clear on every frame by comparing each row's encoded output with the
  previous frame; only changed rows move and are rewritten. On the first
  refresh, clear and draw all rows.
- `Refresh` writes to `os.Stdout`. It must not call `os.Exit`.
- `Size` returns the current terminal size. If it cannot be determined, fall
  back to the canvas size.
- `PollEvent` reads keyboard and mouse events. With a negative timeout it
  blocks; with zero it polls. It returns `(Event{}, false)` on timeout.
  Ctrl-C is delivered as a key event with `Key == 3`, not as a process exit.
- Recognise at least: printable UTF-8, Enter, Backspace, Tab, Escape,
  Ctrl-A..Ctrl-Z, arrows, Home, End, Page Up/Down, Insert, Delete, F1..F12,
  and SGR mouse reports (`\x1b[<b;x;yM` / `m`). Unknown escape sequences are
  ignored.
- Resize events are generated from `SIGWINCH` on Unix. The event carries the
  new `Width` and `Height`.
- The terminal driver must not use cgo or third-party modules. Use `syscall`
  and `unsafe` for termios/size/ioctl on Unix, with build tags. Provide a
  Windows stub that compiles and returns `ErrUnsupported`.

## 11. Quality gates

The deliverable is accepted only when all of these pass:

```sh
gofmt -l .            # no output
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

Additional requirements:

- table-driven tests for attributes, charset, codecs, drawing and transforms
- at least one end-to-end test that renders a canvas through the ANSI encoder
  and compares against a golden string
- no panics for out-of-range coordinates, zero-size canvases, invalid UTF-8,
  malformed native files, or truncated ANSI sequences
- no third-party imports outside the Go standard library
- every `.go` file starts with the WTFPL header
- `README.md` documents the Go-native (non-C-ABI) status and the MVP limits

## 12. Provenance

The implementation follows the WTFPL-licensed core of libcaca. The following
are deliberately excluded and must not be added without a separate licence
review:

- `kernel/multiboot.h` (GPL-2.0-or-later)
- `src/aafire.c`, `examples/demo0.c` (LGPL-2.0-or-later)
- `doc/cacafire.1`
- `caca/mono9.data`, `caca/monobold12.data` (source-font licence unstated)

All Go files carry the WTFPL header. The module has no third-party Go
dependencies. See `PROVENANCE.md` for the module-level statement.
