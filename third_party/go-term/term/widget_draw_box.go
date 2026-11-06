package term

import (
	"math"

	"github.com/go-gui-org/go-gui/gui"
)

// Built-in box drawing.
//
// Box Drawing (U+2500–257F), Block Elements (U+2580–259F) and the Powerline
// separators (U+E0B0–E0BF) are painted as geometry sized to the cell instead
// of as font glyphs. A font's glyph is only as tall as the font, so with
// Cfg.LineHeight above 1 — or with a font whose box glyphs stop short of the
// line height — vertical strokes and blocks leave a gap between rows, and a
// Powerline arrow ends short of the segment background it caps. Geometry
// always reaches the cell edges, so adjacent cells join at any line height.

// Arm weights, one per direction out of the cell centre.
const (
	armNone uint8 = iota
	armLight
	armHeavy
	armDouble
)

// boxArms holds the up, right, down, left weights of each line-drawing
// character, indexed by r-0x2500. A zero entry is drawn by a special case
// (dashes, arcs, diagonals) in drawBoxLine.
var boxArms = [0x80][4]uint8{
	0x00: {0, 1, 0, 1}, // ─
	0x01: {0, 2, 0, 2}, // ━
	0x02: {1, 0, 1, 0}, // │
	0x03: {2, 0, 2, 0}, // ┃
	0x0C: {0, 1, 1, 0}, // ┌
	0x0D: {0, 2, 1, 0}, // ┍
	0x0E: {0, 1, 2, 0}, // ┎
	0x0F: {0, 2, 2, 0}, // ┏
	0x10: {0, 0, 1, 1}, // ┐
	0x11: {0, 0, 1, 2}, // ┑
	0x12: {0, 0, 2, 1}, // ┒
	0x13: {0, 0, 2, 2}, // ┓
	0x14: {1, 1, 0, 0}, // └
	0x15: {1, 2, 0, 0}, // ┕
	0x16: {2, 1, 0, 0}, // ┖
	0x17: {2, 2, 0, 0}, // ┗
	0x18: {1, 0, 0, 1}, // ┘
	0x19: {1, 0, 0, 2}, // ┙
	0x1A: {2, 0, 0, 1}, // ┚
	0x1B: {2, 0, 0, 2}, // ┛
	0x1C: {1, 1, 1, 0}, // ├
	0x1D: {1, 2, 1, 0}, // ┝
	0x1E: {2, 1, 1, 0}, // ┞
	0x1F: {1, 1, 2, 0}, // ┟
	0x20: {2, 1, 2, 0}, // ┠
	0x21: {2, 2, 1, 0}, // ┡
	0x22: {1, 2, 2, 0}, // ┢
	0x23: {2, 2, 2, 0}, // ┣
	0x24: {1, 0, 1, 1}, // ┤
	0x25: {1, 0, 1, 2}, // ┥
	0x26: {2, 0, 1, 1}, // ┦
	0x27: {1, 0, 2, 1}, // ┧
	0x28: {2, 0, 2, 1}, // ┨
	0x29: {2, 0, 1, 2}, // ┩
	0x2A: {1, 0, 2, 2}, // ┪
	0x2B: {2, 0, 2, 2}, // ┫
	0x2C: {0, 1, 1, 1}, // ┬
	0x2D: {0, 1, 1, 2}, // ┭
	0x2E: {0, 2, 1, 1}, // ┮
	0x2F: {0, 2, 1, 2}, // ┯
	0x30: {0, 1, 2, 1}, // ┰
	0x31: {0, 1, 2, 2}, // ┱
	0x32: {0, 2, 2, 1}, // ┲
	0x33: {0, 2, 2, 2}, // ┳
	0x34: {1, 1, 0, 1}, // ┴
	0x35: {1, 1, 0, 2}, // ┵
	0x36: {1, 2, 0, 1}, // ┶
	0x37: {1, 2, 0, 2}, // ┷
	0x38: {2, 1, 0, 1}, // ┸
	0x39: {2, 1, 0, 2}, // ┹
	0x3A: {2, 2, 0, 1}, // ┺
	0x3B: {2, 2, 0, 2}, // ┻
	0x3C: {1, 1, 1, 1}, // ┼
	0x3D: {1, 1, 1, 2}, // ┽
	0x3E: {1, 2, 1, 1}, // ┾
	0x3F: {1, 2, 1, 2}, // ┿
	0x40: {2, 1, 1, 1}, // ╀
	0x41: {1, 1, 2, 1}, // ╁
	0x42: {2, 1, 2, 1}, // ╂
	0x43: {2, 1, 1, 2}, // ╃
	0x44: {2, 2, 1, 1}, // ╄
	0x45: {1, 1, 2, 2}, // ╅
	0x46: {1, 2, 2, 1}, // ╆
	0x47: {2, 2, 1, 2}, // ╇
	0x48: {1, 2, 2, 2}, // ╈
	0x49: {2, 1, 2, 2}, // ╉
	0x4A: {2, 2, 2, 1}, // ╊
	0x4B: {2, 2, 2, 2}, // ╋
	0x50: {0, 3, 0, 3}, // ═
	0x51: {3, 0, 3, 0}, // ║
	0x52: {0, 3, 1, 0}, // ╒
	0x53: {0, 1, 3, 0}, // ╓
	0x54: {0, 3, 3, 0}, // ╔
	0x55: {0, 0, 1, 3}, // ╕
	0x56: {0, 0, 3, 1}, // ╖
	0x57: {0, 0, 3, 3}, // ╗
	0x58: {1, 3, 0, 0}, // ╘
	0x59: {3, 1, 0, 0}, // ╙
	0x5A: {3, 3, 0, 0}, // ╚
	0x5B: {1, 0, 0, 3}, // ╛
	0x5C: {3, 0, 0, 1}, // ╜
	0x5D: {3, 0, 0, 3}, // ╝
	0x5E: {1, 3, 1, 0}, // ╞
	0x5F: {3, 1, 3, 0}, // ╟
	0x60: {3, 3, 3, 0}, // ╠
	0x61: {1, 0, 1, 3}, // ╡
	0x62: {3, 0, 3, 1}, // ╢
	0x63: {3, 0, 3, 3}, // ╣
	0x64: {0, 3, 1, 3}, // ╤
	0x65: {0, 1, 3, 1}, // ╥
	0x66: {0, 3, 3, 3}, // ╦
	0x67: {1, 3, 0, 3}, // ╧
	0x68: {3, 1, 0, 1}, // ╨
	0x69: {3, 3, 0, 3}, // ╩
	0x6A: {1, 3, 1, 3}, // ╪
	0x6B: {3, 1, 3, 1}, // ╫
	0x6C: {3, 3, 3, 3}, // ╬
	0x74: {0, 0, 0, 1}, // ╴
	0x75: {1, 0, 0, 0}, // ╵
	0x76: {0, 1, 0, 0}, // ╶
	0x77: {0, 0, 1, 0}, // ╷
	0x78: {0, 0, 0, 2}, // ╸
	0x79: {2, 0, 0, 0}, // ╹
	0x7A: {0, 2, 0, 0}, // ╺
	0x7B: {0, 0, 2, 0}, // ╻
	0x7C: {0, 2, 0, 1}, // ╼
	0x7D: {1, 0, 2, 0}, // ╽
	0x7E: {0, 1, 0, 2}, // ╾
	0x7F: {2, 0, 1, 0}, // ╿
}

// Directions into a boxArms entry.
const (
	dirUp = iota
	dirRight
	dirDown
	dirLeft
)

// boxPainter paints one cell. Every edge it fills is snapped to the device
// pixel grid, so strokes stay crisp and neighbouring cells meet exactly.
type boxPainter struct {
	t          *Term
	dc         *gui.DrawContext
	x, y, w, h float32
	cx, cy     float32 // cell centre
	lt, ht     float32 // light and heavy stroke thickness
	color      gui.Color
}

// drawBuiltinGlyph paints r into the cell box (x, y, w, h) as geometry and
// reports whether it did. A false return means r is not built in and the
// caller draws the font glyph.
func (t *Term) drawBuiltinGlyph(dc *gui.DrawContext, x, y, w, h float32, r rune, color gui.Color) bool {
	box := r >= 0x2500 && r <= 0x259F
	powerline := r >= 0xE0B0 && r <= 0xE0BF
	if !box && !powerline || w <= 0 || h <= 0 {
		return false
	}
	dp := float32(1)
	if t.draw.pxScale > 0 {
		dp = 1 / t.draw.pxScale
	}
	lt := max(t.snapPx(t.glyphHeight()/14), dp)
	p := boxPainter{
		t: t, dc: dc, x: x, y: y, w: w, h: h,
		cx: x + w/2, cy: y + h/2,
		lt: lt, ht: 2 * lt,
		color: color,
	}
	switch {
	case powerline:
		p.powerline(r)
		return true
	case r >= 0x2580:
		return p.block(r)
	}
	return p.line(r)
}

// fill paints the rectangle [x0, x1) × [y0, y1), snapping each edge and
// widening a span that snapped to nothing to one device pixel.
func (p *boxPainter) fill(x0, y0, x1, y1 float32, c gui.Color) {
	sx0, sx1 := p.t.snapPx(x0), p.t.snapPx(x1)
	sy0, sy1 := p.t.snapPx(y0), p.t.snapPx(y1)
	dp := float32(1)
	if p.t.draw.pxScale > 0 {
		dp = 1 / p.t.draw.pxScale
	}
	if sx1-sx0 < dp {
		sx1 = sx0 + dp
	}
	if sy1-sy0 < dp {
		sy1 = sy0 + dp
	}
	p.dc.FilledRect(sx0, sy0, sx1-sx0, sy1-sy0, c)
}

// hStroke fills a horizontal stroke of thickness th centred on y.
func (p *boxPainter) hStroke(x0, x1, y, th float32) {
	p.fill(x0, y-th/2, x1, y+th/2, p.color)
}

// vStroke fills a vertical stroke of thickness th centred on x.
func (p *boxPainter) vStroke(x, y0, y1, th float32) {
	p.fill(x-th/2, y0, x+th/2, y1, p.color)
}

func (p *boxPainter) thickness(weight uint8) float32 {
	if weight == armHeavy {
		return p.ht
	}
	return p.lt
}

func (p *boxPainter) line(r rune) bool {
	i := r - 0x2500
	switch {
	case i >= 0x04 && i <= 0x0B:
		// ┄┅┆┇ triple dashes, ┈┉┊┋ quadruple; light/heavy alternate and
		// horizontal/vertical alternate in pairs.
		n := 3
		if i >= 0x08 {
			n = 4
		}
		p.dashes(n, (i/2)%2 == 1, i%2 == 1)
		return true
	case i >= 0x4C && i <= 0x4F:
		p.dashes(2, i >= 0x4E, i%2 == 1) // ╌╍╎╏
		return true
	case i >= 0x6D && i <= 0x70:
		p.arc(i - 0x6D) // ╭╮╯╰
		return true
	case i >= 0x71 && i <= 0x73:
		if i != 0x72 {
			p.diagonal(true) // ╱ and half of ╳
		}
		if i != 0x71 {
			p.diagonal(false) // ╲ and half of ╳
		}
		return true
	}
	arms := boxArms[i]
	if arms == [4]uint8{} {
		return false
	}
	p.arms(arms)
	return true
}

// dashes draws n evenly spaced segments across the cell, vertical or
// horizontal, light or heavy.
func (p *boxPainter) dashes(n int, vertical, heavy bool) {
	th := p.lt
	if heavy {
		th = p.ht
	}
	span := p.w
	if vertical {
		span = p.h
	}
	step := span / float32(n)
	gap := step * 0.3
	for k := range n {
		a := float32(k)*step + gap/2
		b := float32(k+1)*step - gap/2
		if vertical {
			p.vStroke(p.cx, p.y+a, p.y+b, th)
		} else {
			p.hStroke(p.x+a, p.x+b, p.cy, th)
		}
	}
}

// arms draws a line-drawing character from its four arm weights.
//
// Light and heavy arms run from the edge to the centre, overlapping the
// perpendicular stroke so the join is solid. A double arm is two light strokes
// offset by ±d; where it meets other doubles each stroke stops at the near or
// far perpendicular stroke so corners and tees close the way fonts draw them
// (╔ ╦ ╬). A single arm meeting a double pair stops on the near stroke of a
// tee (╟) and runs to the far stroke of a corner (╓).
func (p *boxPainter) arms(a [4]uint8) {
	d := p.lt
	up, right, down, left := a[dirUp], a[dirRight], a[dirDown], a[dirLeft]

	// Thickness of the solid vertical / horizontal stroke through the
	// centre, which single arms of the other axis overlap.
	var vt, ht float32
	for _, w := range []uint8{up, down} {
		if w == armLight || w == armHeavy {
			vt = max(vt, p.thickness(w))
		}
	}
	for _, w := range []uint8{left, right} {
		if w == armLight || w == armHeavy {
			ht = max(ht, p.thickness(w))
		}
	}

	// singleEnd is how far past the centre (toward the opposite side) a
	// single arm reaches. perpA/perpB are the perpendicular arms, opp the
	// opposite one, solid the solid perpendicular thickness.
	singleEnd := func(th float32, perpA, perpB, opp uint8, solid float32) float32 {
		da, db := perpA == armDouble, perpB == armDouble
		switch {
		case da && db && opp == armNone:
			return -d + p.lt/2 // tee into a double pair: near stroke
		case (da || db) && opp == armNone:
			return d + p.lt/2 // corner into a double pair: far stroke
		case solid > 0:
			return solid / 2
		default:
			return th / 2
		}
	}
	// doubleEnd is the same for one stroke of a double arm: near is the
	// perpendicular arm on this stroke's side, far the other one.
	doubleEnd := func(near, far uint8, solid float32) float32 {
		switch {
		case near == armDouble:
			return -d + p.lt/2
		case far == armDouble:
			return d + p.lt/2
		case solid > 0:
			return solid / 2
		default:
			return p.lt / 2
		}
	}

	switch right {
	case armLight, armHeavy:
		th := p.thickness(right)
		e := p.cx - singleEnd(th, up, down, left, vt)
		p.hStroke(e, p.x+p.w, p.cy, th)
	case armDouble:
		p.hStroke(p.cx-doubleEnd(up, down, vt), p.x+p.w, p.cy-d, p.lt)
		p.hStroke(p.cx-doubleEnd(down, up, vt), p.x+p.w, p.cy+d, p.lt)
	}
	switch left {
	case armLight, armHeavy:
		th := p.thickness(left)
		e := p.cx + singleEnd(th, up, down, right, vt)
		p.hStroke(p.x, e, p.cy, th)
	case armDouble:
		p.hStroke(p.x, p.cx+doubleEnd(up, down, vt), p.cy-d, p.lt)
		p.hStroke(p.x, p.cx+doubleEnd(down, up, vt), p.cy+d, p.lt)
	}
	switch down {
	case armLight, armHeavy:
		th := p.thickness(down)
		e := p.cy - singleEnd(th, left, right, up, ht)
		p.vStroke(p.cx, e, p.y+p.h, th)
	case armDouble:
		p.vStroke(p.cx-d, p.cy-doubleEnd(left, right, ht), p.y+p.h, p.lt)
		p.vStroke(p.cx+d, p.cy-doubleEnd(right, left, ht), p.y+p.h, p.lt)
	}
	switch up {
	case armLight, armHeavy:
		th := p.thickness(up)
		e := p.cy + singleEnd(th, left, right, down, ht)
		p.vStroke(p.cx, p.y, e, th)
	case armDouble:
		p.vStroke(p.cx-d, p.y, p.cy+doubleEnd(left, right, ht), p.lt)
		p.vStroke(p.cx+d, p.y, p.cy+doubleEnd(right, left, ht), p.lt)
	}
}

// arc draws a rounded corner: 0 ╭, 1 ╮, 2 ╯, 3 ╰. A quarter circle of
// radius half the cell width joins the centre lines, and a straight stroke
// carries the vertical end to the cell edge when the cell is taller than wide.
func (p *boxPainter) arc(k rune) {
	rad := min(p.w, p.h) / 2
	// sx/sy point from the centre toward the two edges the arc reaches.
	sx, sy := float32(1), float32(1)
	switch k {
	case 1:
		sx = -1
	case 2:
		sx, sy = -1, -1
	case 3:
		sy = -1
	}
	// Arc centre sits diagonally off the cell centre toward both edges.
	ax, ay := p.cx+sx*rad, p.cy+sy*rad
	in, out := rad-p.lt/2, rad+p.lt/2
	const segs = 12
	var pts [8]float32
	for s := range segs {
		a0 := float64(s) / segs * math.Pi / 2
		a1 := float64(s+1) / segs * math.Pi / 2
		// Angle 0 points back at the cell centre's column (−sx), angle π/2
		// back at its row (−sy).
		c0, s0 := float32(math.Cos(a0)), float32(math.Sin(a0))
		c1, s1 := float32(math.Cos(a1)), float32(math.Sin(a1))
		pts = [8]float32{
			ax - sx*in*c0, ay - sy*in*s0,
			ax - sx*out*c0, ay - sy*out*s0,
			ax - sx*out*c1, ay - sy*out*s1,
			ax - sx*in*c1, ay - sy*in*s1,
		}
		p.dc.FilledPolygon(pts[:], p.color)
	}
	// Straight remainder of the vertical arm, from the arc's end to the edge.
	// The arm along the cell's shorter side has no remainder; skip it rather
	// than let fill widen an empty span into the neighbouring cell.
	if sy > 0 && ay < p.y+p.h {
		p.vStroke(p.cx, ay, p.y+p.h, p.lt)
	} else if sy < 0 && ay > p.y {
		p.vStroke(p.cx, p.y, ay, p.lt)
	}
	if sx > 0 && ax < p.x+p.w {
		p.hStroke(ax, p.x+p.w, p.cy, p.lt)
	} else if sx < 0 && ax > p.x {
		p.hStroke(p.x, ax, p.cy, p.lt)
	}
}

// diagonal draws ╱ (rising) or ╲ corner to corner.
func (p *boxPainter) diagonal(rising bool) {
	// Horizontal half-width that gives the stroke a perpendicular thickness
	// of lt.
	a := p.lt * float32(math.Hypot(float64(p.w), float64(p.h))) / (2 * p.h)
	x0, x1 := p.x, p.x+p.w
	top, bot := p.y, p.y+p.h
	if rising {
		p.dc.FilledPolygon([]float32{x0 - a, bot, x0 + a, bot, x1 + a, top, x1 - a, top}, p.color)
	} else {
		p.dc.FilledPolygon([]float32{x0 - a, top, x0 + a, top, x1 + a, bot, x1 - a, bot}, p.color)
	}
}

// Quadrant bits for the U+2596–259F block elements.
const (
	quadUL = 1 << iota
	quadUR
	quadLL
	quadLR
)

var blockQuads = [10]uint8{
	quadLL,                   // ▖
	quadLR,                   // ▗
	quadUL,                   // ▘
	quadUL | quadLL | quadLR, // ▙
	quadUL | quadLR,          // ▚
	quadUL | quadUR | quadLL, // ▛
	quadUL | quadUR | quadLR, // ▜
	quadUR,                   // ▝
	quadUR | quadLL,          // ▞
	quadUR | quadLL | quadLR, // ▟
}

func (p *boxPainter) block(r rune) bool {
	x0, y0, x1, y1 := p.x, p.y, p.x+p.w, p.y+p.h
	switch {
	case r == 0x2580: // ▀
		p.fill(x0, y0, x1, p.cy, p.color)
	case r >= 0x2581 && r <= 0x2588: // ▁ … █, lower n/8
		n := float32(r - 0x2580)
		p.fill(x0, y1-p.h*n/8, x1, y1, p.color)
	case r >= 0x2589 && r <= 0x258F: // ▉ … ▏, left n/8
		n := float32(0x2590 - r)
		p.fill(x0, y0, x0+p.w*n/8, y1, p.color)
	case r == 0x2590: // ▐
		p.fill(p.cx, y0, x1, y1, p.color)
	case r >= 0x2591 && r <= 0x2593: // ░ ▒ ▓
		p.fill(x0, y0, x1, y1, p.color.WithOpacity(float32(r-0x2590)/4))
	case r == 0x2594: // ▔
		p.fill(x0, y0, x1, y0+p.h/8, p.color)
	case r == 0x2595: // ▕
		p.fill(x1-p.w/8, y0, x1, y1, p.color)
	default: // ▖ … ▟
		q := blockQuads[r-0x2596]
		if q&quadUL != 0 {
			p.fill(x0, y0, p.cx, p.cy, p.color)
		}
		if q&quadUR != 0 {
			p.fill(p.cx, y0, x1, p.cy, p.color)
		}
		if q&quadLL != 0 {
			p.fill(x0, p.cy, p.cx, y1, p.color)
		}
		if q&quadLR != 0 {
			p.fill(p.cx, p.cy, x1, y1, p.color)
		}
	}
	return true
}

// powerline draws the U+E0B0–E0BF separators: solid and thin arrows, rounded
// caps, and the slanted triangles and slashes. Solid shapes run edge to edge so
// they meet the neighbouring segment's background without a seam.
func (p *boxPainter) powerline(r rune) {
	x0, y0, x1, y1 := p.x, p.y, p.x+p.w, p.y+p.h
	switch r {
	case 0xE0B0: // right-pointing solid arrow
		p.poly(x0, y0, x1, p.cy, x0, y1)
	case 0xE0B1: // right-pointing thin arrow
		p.segment(x0, y0, x1, p.cy)
		p.segment(x1, p.cy, x0, y1)
	case 0xE0B2: // left-pointing solid arrow
		p.poly(x1, y0, x1, y1, x0, p.cy)
	case 0xE0B3: // left-pointing thin arrow
		p.segment(x1, y0, x0, p.cy)
		p.segment(x0, p.cy, x1, y1)
	case 0xE0B4: // right half circle, solid
		p.halfEllipse(x0, true, true)
	case 0xE0B5: // right half circle, thin
		p.halfEllipse(x0, true, false)
	case 0xE0B6: // left half circle, solid
		p.halfEllipse(x1, false, true)
	case 0xE0B7: // left half circle, thin
		p.halfEllipse(x1, false, false)
	case 0xE0B8: // lower-left triangle
		p.poly(x0, y0, x1, y1, x0, y1)
	case 0xE0BA: // lower-right triangle
		p.poly(x1, y0, x1, y1, x0, y1)
	case 0xE0BC: // upper-left triangle
		p.poly(x0, y0, x1, y0, x0, y1)
	case 0xE0BE: // upper-right triangle
		p.poly(x0, y0, x1, y0, x1, y1)
	case 0xE0B9, 0xE0BF: // backslash separator
		p.diagonal(false)
	case 0xE0BB, 0xE0BD: // forward-slash separator
		p.diagonal(true)
	}
}

// poly fills a convex polygon given as flat x, y pairs.
func (p *boxPainter) poly(pts ...float32) {
	p.dc.FilledPolygon(pts, p.color)
}

// segment strokes a light line from (ax, ay) to (bx, by), offsetting the
// quad along the line's normal so its thickness is lt at any angle.
func (p *boxPainter) segment(ax, ay, bx, by float32) {
	dx, dy := bx-ax, by-ay
	l := float32(math.Hypot(float64(dx), float64(dy)))
	if l == 0 {
		return
	}
	nx, ny := -dy/l*p.lt/2, dx/l*p.lt/2
	p.poly(ax+nx, ay+ny, bx+nx, by+ny, bx-nx, by-ny, ax-nx, ay-ny)
}

// halfEllipse draws a half ellipse whose flat side lies on the vertical line
// x = edge, bulging right (toward +x) or left across the full cell, solid or
// as a light outline.
func (p *boxPainter) halfEllipse(edge float32, right, solid bool) {
	const segs = 24
	sx := float32(1)
	if !right {
		sx = -1
	}
	rx, ry := p.w, p.h/2
	pt := func(rx, ry float32, k int) (float32, float32) {
		// k = 0 is the top of the flat side, k = segs the bottom.
		a := math.Pi * float64(k) / segs
		return edge + sx*rx*float32(math.Sin(a)), p.cy - ry*float32(math.Cos(a))
	}
	if solid {
		pts := make([]float32, 0, 2*(segs+1))
		for k := 0; k <= segs; k++ {
			x, y := pt(rx, ry, k)
			pts = append(pts, x, y)
		}
		p.poly(pts...)
		return
	}
	// Outline: the inner edge is pulled in by lt so the stroke stays inside
	// the cell.
	irx, iry := rx-p.lt, ry-p.lt/2
	for k := range segs {
		ox0, oy0 := pt(rx, ry, k)
		ox1, oy1 := pt(rx, ry, k+1)
		ix1, iy1 := pt(irx, iry, k+1)
		ix0, iy0 := pt(irx, iry, k)
		p.poly(ox0, oy0, ox1, oy1, ix1, iy1, ix0, iy0)
	}
}
