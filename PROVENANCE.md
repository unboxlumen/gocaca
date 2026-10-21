# Provenance

gocaca (`github.com/unboxlumen/gocaca`) is released under the WTFPL v2. It is a
Go port of the **core** of libcaca — the `caca/` library sources at
`https://github.com/cacalabs/libcaca` (reference commit
`de48e1a0888caabdc6276afdea46e9096d0adb05`) — and that core is distributed
under the WTFPL v2, as recorded in the upstream `COPYING` file. Only that
WTFPL-licensed core was used; the GPL/LGPL files and the unlicensed font data
elsewhere in the upstream tree are not part of this module.

## Ported

The behaviour of these C modules was used as the reference for the Go
implementation:

- `caca/attr.c`, `caca/charset.c`, `caca/string.c`, `caca/canvas.c`
- `caca/frame.c`, `caca/dirty.c`
- the drawing primitives corresponding to `caca/line.c`, `caca/box.c`,
  `caca/conic.c`, `caca/triangle.c`
- `caca/transform.c`
- the text/ANSI/native codec behaviour from `caca/codec/`
- the dithering behaviour of `caca/dither.c` (algorithms, colour modes,
  charsets, palette and error diffusion), reimplemented in Go
- the attribute encoding and palette tables from `caca/attr.c`, including
  the 14-bit component layout

## Deliberately excluded

These parts are not included in this module:

- `kernel/multiboot.h` — GPL-2.0-or-later
- `src/aafire.c`, `examples/demo0.c` — LGPL-2.0-or-later
- `doc/cacafire.1` — documents the LGPL cacafire demo
- `caca/mono9.data`, `caca/monobold12.data` — generated font data whose
  source-font licence is not stated; no font data is copied
- bitmap fonts, FIGfont, GUI drivers, C ABI bindings and the
  platform-specific C drivers

The WTFPL permits relicensing and derivative works. All Go source in this module is released under WTFPL v2; see `LICENSE`. The Go standard library is
distributed under its own BSD-style licence and is not part of this module's
source.

## Licence audit

- All new Go files carry the WTFPL v2 header.
- The implementation follows the WTFPL-licensed core; no GPL or LGPL source
  was used.
- No generated font data was copied. The CP437 mapping and ANSI palette are
  factual tables; the ANSI, UTF-8, and native codecs implement public formats.
- `attr_test.go` and `attr_golden_test.go` contain reference values produced
  by a C build of libcaca; they are test expectations, not copied code.
- The module has no third-party Go dependencies.
