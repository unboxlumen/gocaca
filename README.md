# libcaca (Go)

A pure Go colour ASCII-art canvas library, ported from the WTFPL-licensed
core of libcaca. The package is **stdlib-only**: no cgo and no third-party
modules. It is Go-native; it does not export a C ABI or `libcaca.so`.

## Status

Implemented:

- canvas, per-cell characters and attributes
- multiple frames, handles, dirty rectangles
- ANSI and ARGB4444 attributes, bit-compatible with libcaca, with
  bold/italics/underline/blink
- UTF-8, UTF-32, CP437 and ASCII helpers, fullwidth cells
- line, thin line, box, circle, ellipse and triangle drawing
- invert, flip, flop, rotation and stretch transforms
- `text`, `ansi`, `utf8` and native `caca` codecs; `.caca` files are
  byte-compatible with libcaca
- ANSI terminal driver with raw mode, resize events, keyboard and SGR mouse
  input on Linux and macOS
- image dithering with six algorithms, seven colour modes and three charsets

Not yet ported: bitmap fonts, FIGfont, HTML/SVG/TGA/PS/troff/IRC exports,
GUI drivers, C ABI bindings, and the platform-specific C drivers. The
generated font data is deliberately excluded because its source-font licence
is not stated.

## Install

```sh
go get github.com/unboxlumen/libcaca
```

## Quick start

```go
package main

import (
    "log"

    "github.com/unboxlumen/libcaca"
)

func main() {
    c, err := libcaca.NewCanvas(40, 10)
    if err != nil {
        log.Fatal(err)
    }
    if err := c.SetColor(libcaca.White, libcaca.Blue); err != nil {
        log.Fatal(err)
    }
    c.PutString(2, 2, "hello, caca")
    c.DrawThinBox(0, 0, 40, 10)

    t, err := libcaca.NewTerminal(c)
    if err != nil {
        log.Fatal(err)
    }
    defer t.Close()
    if err := t.Refresh(); err != nil {
        log.Fatal(err)
    }
}
```

`ansi` is CP437 for compatibility with classic ANSI art. Use `utf8` when
writing to a modern UTF-8 terminal if you want Unicode box drawing and
fullwidth characters to survive.

## Demos

From the module root:

```sh
go run ./cmd/libcaca-demo           # animated canvas demo
go run ./cmd/libcaca-pic image.png  # render an image into the terminal
```

`libcaca-demo` animates a bouncing ball and handles terminal resize.
`libcaca-pic` accepts PNG, JPEG and GIF and supports:

```sh
go run ./cmd/libcaca-pic -width 100 -charset shades -color full16 \
    -dither fstein -brightness 1.1 photo.jpg
```

| flag | values |
|---|---|
| `-charset` | `ascii`, `shades`, `blocks` |
| `-color` | `mono`, `gray`, `8`, `16`, `fullgray`, `full8`, `full16` |
| `-dither` | `none`, `ordered2`, `ordered4`, `ordered8`, `random`, `fstein` |
| `-width` | cells, `0` = terminal width |
| `-height` | cells, `0` = aspect ratio |
| `-brightness`, `-gamma`, `-contrast` | floats |

Both demos quit on `q`, `Q`, `Ctrl-C`, or a terminal close. When stdout is
not a terminal they print one UTF-8 ANSI frame and exit.

## Quality gates

From the module root:

```sh
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

## Licence

WTFPL v2. See `LICENSE`. Every source file carries the same grant. The Go
standard library is distributed under its own BSD-style licence; it is not
part of this module's source.

See `PROVENANCE.md` for the ported modules and the deliberately excluded
GPL/LGPL files.
