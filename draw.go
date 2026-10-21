// Copyright (c) 2026 The libcaca Go Authors.
// This program is free software. It comes without any warranty, to the
// extent permitted by applicable law. You can redistribute it and/or
// modify it under the terms of the Do What The Fuck You Want To Public
// License, Version 2, as published by Sam Hocevar.

package libcaca

import "math"

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// DrawLine draws a Bresenham line, including both endpoints.
func (c *Canvas) DrawLine(x1, y1, x2, y2 int, ch rune) {
	dx := iabs(x2 - x1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	dy := -iabs(y2 - y1)
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx + dy
	for {
		c.PutChar(x1, y1, ch)
		if x1 == x2 && y1 == y2 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
}

// DrawPolyline draws consecutive line segments.
func (c *Canvas) DrawPolyline(points []Point, ch rune) {
	if len(points) == 0 {
		return
	}
	if len(points) == 1 {
		c.PutChar(points[0].X, points[0].Y, ch)
		return
	}
	for i := 1; i < len(points); i++ {
		c.DrawLine(points[i-1].X, points[i-1].Y, points[i].X, points[i].Y, ch)
	}
}

// DrawThinLine draws a single-cell line using box-drawing characters.
func (c *Canvas) DrawThinLine(x1, y1, x2, y2 int) {
	switch {
	case y1 == y2:
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		for x := x1; x <= x2; x++ {
			c.PutChar(x, y1, '─')
		}
	case x1 == x2:
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		for y := y1; y <= y2; y++ {
			c.PutChar(x1, y, '│')
		}
	default:
		ch := '\\'
		if (x1 < x2) != (y1 < y2) {
			ch = '/'
		}
		c.DrawLine(x1, y1, x2, y2, ch)
	}
}

// DrawThinPolyline draws consecutive thin segments.
func (c *Canvas) DrawThinPolyline(points []Point) {
	for i := 1; i < len(points); i++ {
		c.DrawThinLine(points[i-1].X, points[i-1].Y, points[i].X, points[i].Y)
	}
}

// DrawBox draws a rectangle outline using ch.
func (c *Canvas) DrawBox(x, y, width, height int, ch rune) {
	if width <= 0 || height <= 0 {
		return
	}
	c.DrawLine(x, y, x+width-1, y, ch)
	c.DrawLine(x, y+height-1, x+width-1, y+height-1, ch)
	c.DrawLine(x, y, x, y+height-1, ch)
	c.DrawLine(x+width-1, y, x+width-1, y+height-1, ch)
}

// FillBox fills a rectangle with ch.
func (c *Canvas) FillBox(x, y, width, height int, ch rune) {
	if width <= 0 || height <= 0 {
		return
	}
	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			c.PutChar(x+i, y+j, ch)
		}
	}
}

// DrawThinBox draws a single-cell box with rounded corners.
func (c *Canvas) DrawThinBox(x, y, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	if width == 1 || height == 1 {
		c.FillBox(x, y, width, height, '─')
		return
	}
	c.PutChar(x, y, '┌')
	c.PutChar(x+width-1, y, '┐')
	c.PutChar(x, y+height-1, '└')
	c.PutChar(x+width-1, y+height-1, '┘')
	for i := 1; i < width-1; i++ {
		c.PutChar(x+i, y, '─')
		c.PutChar(x+i, y+height-1, '─')
	}
	for j := 1; j < height-1; j++ {
		c.PutChar(x, y+j, '│')
		c.PutChar(x+width-1, y+j, '│')
	}
}

// DrawCircle draws a circle outline.
func (c *Canvas) DrawCircle(cx, cy, radius int, ch rune) {
	c.DrawEllipse(cx, cy, radius, radius, ch)
}

// DrawEllipse draws an ellipse outline.
func (c *Canvas) DrawEllipse(cx, cy, rx, ry int, ch rune) {
	if rx < 0 {
		rx = -rx
	}
	if ry < 0 {
		ry = -ry
	}
	if rx == 0 && ry == 0 {
		c.PutChar(cx, cy, ch)
		return
	}
	if rx == 0 {
		c.DrawLine(cx, cy-ry, cx, cy+ry, ch)
		return
	}
	if ry == 0 {
		c.DrawLine(cx-rx, cy, cx+rx, cy, ch)
		return
	}

	rx2, ry2 := int64(rx)*int64(rx), int64(ry)*int64(ry)
	x, y := int64(0), int64(ry)
	d1 := ry2 - rx2*int64(ry) + rx2/4
	for 2*ry2*x < 2*rx2*y {
		c.plotEllipse(cx, cy, int(x), int(y), ch)
		if d1 < 0 {
			d1 += 2*ry2*x + 3*ry2
		} else {
			d1 += 2*ry2*x - 2*rx2*y + 5*ry2
			y--
		}
		x++
	}
	d2 := float64(ry2)*(float64(x)+0.5)*(float64(x)+0.5) +
		float64(rx2)*float64(y-1)*float64(y-1) - float64(rx2*ry2)
	for y >= 0 {
		c.plotEllipse(cx, cy, int(x), int(y), ch)
		if d2 > 0 {
			d2 += float64(-2*rx2*y + 3*rx2)
			y--
		} else {
			d2 += float64(2*ry2*x - 2*rx2*y + 5*rx2)
			x++
			y--
		}
	}
}

func (c *Canvas) plotEllipse(cx, cy, x, y int, ch rune) {
	c.PutChar(cx+x, cy+y, ch)
	c.PutChar(cx-x, cy+y, ch)
	c.PutChar(cx+x, cy-y, ch)
	c.PutChar(cx-x, cy-y, ch)
}

// FillEllipse fills an ellipse.
func (c *Canvas) FillEllipse(cx, cy, rx, ry int, ch rune) {
	if rx < 0 {
		rx = -rx
	}
	if ry < 0 {
		ry = -ry
	}
	if ry == 0 {
		c.DrawLine(cx-rx, cy, cx+rx, cy, ch)
		return
	}
	for dy := -ry; dy <= ry; dy++ {
		t := 1.0 - float64(dy*dy)/float64(ry*ry)
		if t < 0 {
			continue
		}
		dx := int(math.Sqrt(float64(rx*rx) * t))
		c.DrawLine(cx-dx, cy+dy, cx+dx, cy+dy, ch)
	}
}

// DrawTriangle draws the three edges of a triangle.
func (c *Canvas) DrawTriangle(x1, y1, x2, y2, x3, y3 int, ch rune) {
	c.DrawLine(x1, y1, x2, y2, ch)
	c.DrawLine(x2, y2, x3, y3, ch)
	c.DrawLine(x3, y3, x1, y1, ch)
}

// FillTriangle fills a triangle with a scanline rasteriser. Boundary cells
// are included, which gives a solid shape in a character grid.
func (c *Canvas) FillTriangle(x1, y1, x2, y2, x3, y3 int, ch rune) {
	minY, maxY := min(y1, min(y2, y3)), max(y1, max(y2, y3))
	edges := [3][4]int{{x1, y1, x2, y2}, {x2, y2, x3, y3}, {x3, y3, x1, y1}}
	for y := minY; y <= maxY; y++ {
		sy := float64(y) + 0.5
		var xs []float64
		for _, e := range edges {
			ax, ay, bx, by := e[0], e[1], e[2], e[3]
			if ay == by {
				continue
			}
			lo, hi := math.Min(float64(ay), float64(by)), math.Max(float64(ay), float64(by))
			if sy < lo || sy >= hi {
				continue
			}
			t := (sy - float64(ay)) / float64(by-ay)
			xs = append(xs, float64(ax)+t*float64(bx-ax))
		}
		if len(xs) < 2 {
			continue
		}
		sortFloat64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			start := int(math.Floor(xs[i]))
			end := int(math.Ceil(xs[i+1])) - 1
			for x := start; x <= end; x++ {
				c.PutChar(x, y, ch)
			}
		}
	}
}

func sortFloat64s(v []float64) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
