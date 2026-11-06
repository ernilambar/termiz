package term

import (
	"testing"

	"github.com/go-gui-org/go-gui/gui"
)

// boxCell is the test cell box: 10×30 at the origin, tall like a row with
// line height above 1, on a 2× display.
const boxW, boxH = 10, 30

func drawBox(t *testing.T, r rune) (*gui.DrawContext, bool) {
	t.Helper()
	tm := &Term{cellW: boxW, cellH: boxH, glyphH: 20}
	tm.draw.pxScale = 2
	dc := gui.NewDrawContext(boxW, boxH, nil)
	ok := tm.drawBuiltinGlyph(dc, 0, 0, boxW, boxH, r, gui.RGB(255, 255, 255))
	return dc, ok
}

// covered reports whether any filled triangle contains (px, py).
func covered(dc *gui.DrawContext, px, py float32) bool {
	for _, b := range dc.Batches() {
		tr := b.Triangles
		for i := 0; i+5 < len(tr); i += 6 {
			if inTriangle(px, py, tr[i], tr[i+1], tr[i+2], tr[i+3], tr[i+4], tr[i+5]) {
				return true
			}
		}
	}
	return false
}

func inTriangle(px, py, ax, ay, bx, by, cx, cy float32) bool {
	d1 := (px-bx)*(ay-by) - (ax-bx)*(py-by)
	d2 := (px-cx)*(by-cy) - (bx-cx)*(py-cy)
	d3 := (px-ax)*(cy-ay) - (cx-ax)*(py-ay)
	neg := d1 < 0 || d2 < 0 || d3 < 0
	pos := d1 > 0 || d2 > 0 || d3 > 0
	return !(neg && pos)
}

// Every table-driven character must reach exactly the cell edges its arms
// name, so it joins its neighbours at any row height. Double arms are offset
// from the centre line, so their edge probe is the centre ± the offset.
func TestBuiltinBoxArmsReachEdges(t *testing.T) {
	const lt = 1.5 // snapPx(20/14) at 2×
	for i, arms := range boxArms {
		if arms == [4]uint8{} {
			continue
		}
		r := rune(0x2500 + i)
		dc, ok := drawBox(t, r)
		if !ok {
			t.Errorf("%c: not built in", r)
			continue
		}
		probes := [4][2]float32{
			dirUp:    {boxW / 2, 0.25},
			dirRight: {boxW - 0.25, boxH / 2},
			dirDown:  {boxW / 2, boxH - 0.25},
			dirLeft:  {0.25, boxH / 2},
		}
		for dir, p := range probes {
			want := arms[dir] != armNone
			hit := covered(dc, p[0], p[1])
			if arms[dir] == armDouble {
				// Centre must be empty between the two strokes; each stroke
				// sits at ±lt.
				if dir == dirUp || dir == dirDown {
					hit = covered(dc, p[0]-lt, p[1]) && covered(dc, p[0]+lt, p[1]) && !hit
				} else {
					hit = covered(dc, p[0], p[1]-lt) && covered(dc, p[0], p[1]+lt) && !hit
				}
			}
			if hit != want {
				t.Errorf("%c: arm %d at edge = %v, want %v", r, dir, hit, want)
			}
		}
	}
}

func TestBuiltinBoxSpecialLines(t *testing.T) {
	for _, r := range []rune{'┄', '┆', '┈', '┊', '╌', '╎', '╭', '╮', '╯', '╰', '╱', '╲', '╳'} {
		if dc, ok := drawBox(t, r); !ok || len(dc.Batches()) == 0 {
			t.Errorf("%c: drawn = %v, batches = %d", r, ok, len(dc.Batches()))
		}
	}
	// ╭ joins the cell's bottom and right edges and leaves the others bare.
	dc, _ := drawBox(t, '╭')
	if !covered(dc, boxW/2, boxH-0.25) || !covered(dc, boxW-0.25, boxH/2) {
		t.Error("╭ does not reach the bottom and right edges")
	}
	if covered(dc, boxW/2, 0.25) || covered(dc, 0.25, boxH/2) {
		t.Error("╭ reaches the top or left edge")
	}
}

// Blocks fill the full row height, which is the point of building them in.
func TestBuiltinBlocks(t *testing.T) {
	cases := []struct {
		r          rune
		top, bot   bool // left-column samples near the top and bottom edge
		left, rite bool // mid-row samples near the left and right edge
	}{
		{'█', true, true, true, true},
		{'▀', true, false, false, false},
		{'▄', false, true, true, true},
		{'▌', true, true, true, false},
		{'▐', false, false, false, true},
		{'▘', true, false, false, false},
		{'▟', false, true, true, true},
	}
	for _, c := range cases {
		dc, ok := drawBox(t, c.r)
		if !ok {
			t.Errorf("%c: not built in", c.r)
			continue
		}
		got := [4]bool{
			covered(dc, 1, 0.25), covered(dc, 1, boxH-0.25),
			covered(dc, 0.25, boxH*0.75), covered(dc, boxW-0.25, boxH*0.75),
		}
		want := [4]bool{c.top, c.bot, c.left, c.rite}
		if got != want {
			t.Errorf("%c: top/bottom/left/right = %v, want %v", c.r, got, want)
		}
	}
}

func TestBuiltinGlyphSkipsOthers(t *testing.T) {
	for _, r := range []rune{'A', '■', 0x25FF, 0x24FF, 0xE0AF, 0xE0C0} {
		if _, ok := drawBox(t, r); ok {
			t.Errorf("%U drawn as built-in", r)
		}
	}
}

// The draw pass routes box characters to geometry, not text.
func TestOnDraw_BoxCharIsGeometry(t *testing.T) {
	tm, dc := newDrawTerm(4, 8, 10, 20)
	tm.grid.Mu.Lock()
	tm.grid.Put('│')
	tm.grid.Mu.Unlock()
	tm.onDraw(dc)
	for _, tx := range dc.Texts() {
		if tx.Text == "│" {
			t.Fatal("│ drawn as font text")
		}
	}
}

// Solid Powerline caps fill their flat side edge to edge, so they meet the
// segment background at any row height, and leave the far corners empty.
func TestBuiltinPowerline(t *testing.T) {
	const top, bot, mid = 0.25, boxH - 0.25, boxH / 2
	const l, r = 0.25, boxW - 0.25
	cases := []struct {
		r    rune
		in   [][2]float32
		out  [][2]float32
		name string
	}{
		{0xE0B0, [][2]float32{{l, 1}, {l, boxH - 1}, {r, mid}}, [][2]float32{{r, top}, {r, bot}}, "right arrow"},
		{0xE0B2, [][2]float32{{r, 1}, {r, boxH - 1}, {l, mid}}, [][2]float32{{l, top}, {l, bot}}, "left arrow"},
		{0xE0B4, [][2]float32{{l, top}, {l, bot}, {r, mid}}, [][2]float32{{r, top}, {r, bot}}, "right half circle"},
		{0xE0B6, [][2]float32{{r, top}, {r, bot}, {l, mid}}, [][2]float32{{l, top}, {l, bot}}, "left half circle"},
		{0xE0B8, [][2]float32{{l, bot}, {l, mid}}, [][2]float32{{r, top}}, "lower-left triangle"},
		{0xE0BE, [][2]float32{{r, top}, {r, mid}}, [][2]float32{{l, bot}}, "upper-right triangle"},
		{0xE0B1, [][2]float32{{boxW / 2, boxH / 4}}, [][2]float32{{l, mid}, {r, top}}, "thin right arrow"},
	}
	for _, c := range cases {
		dc, ok := drawBox(t, c.r)
		if !ok {
			t.Errorf("%s: not built in", c.name)
			continue
		}
		for _, pt := range c.in {
			if !covered(dc, pt[0], pt[1]) {
				t.Errorf("%s: (%v, %v) empty, want filled", c.name, pt[0], pt[1])
			}
		}
		for _, pt := range c.out {
			if covered(dc, pt[0], pt[1]) {
				t.Errorf("%s: (%v, %v) filled, want empty", c.name, pt[0], pt[1])
			}
		}
	}
	for r := rune(0xE0B0); r <= 0xE0BF; r++ {
		if dc, ok := drawBox(t, r); !ok || len(dc.Batches()) == 0 {
			t.Errorf("%U: drawn = %v, batches = %d", r, ok, len(dc.Batches()))
		}
	}
}
