package term

import (
	"testing"

	"github.com/go-gui-org/go-gui/gui"
)

func TestEffectiveLineHeight(t *testing.T) {
	for _, tc := range []struct{ in, want float32 }{
		{0, 1}, {0.5, 1}, {1, 1}, {1.25, 1.25}, {MaxLineHeight, MaxLineHeight}, {3, 1},
	} {
		if got := effectiveLineHeight(tc.in); got != tc.want {
			t.Errorf("effectiveLineHeight(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// A 1.5 line height on a 20px font makes 30px rows with each glyph centred,
// 5px below the row top.
func TestOnDraw_LineHeightCentresGlyphs(t *testing.T) {
	g := newGrid(4, 8)
	tm := &Term{grid: g, cmd: syncScheduler{}}
	tm.cfg.LineHeight = 1.5
	tm.focused.Store(true)
	tm.mouse.hoverR.Store(-1)
	tm.mouse.hoverC.Store(-1)
	g.Mu.Lock()
	g.CursorR, g.CursorC = 1, 0
	g.Put('A')
	g.Mu.Unlock()

	dc := gui.NewDrawContext(80, 120, testTextMeasurer{cellW: 10, cellH: 20})
	tm.onDraw(dc)

	if tm.cellH != 30 {
		t.Fatalf("cellH = %v, want 30", tm.cellH)
	}
	for _, tx := range dc.Texts() {
		if tx.Text == "A" {
			if tx.Y != 35 {
				t.Errorf("glyph Y = %v, want 35 (row 1 at 30 + 5)", tx.Y)
			}
			return
		}
	}
	t.Fatal("glyph not drawn")
}

func TestSetLineHeightRemeasures(t *testing.T) {
	tm, _ := newDrawTerm(4, 8, 10, 20)
	tm.SetLineHeight(1.2)
	if tm.cellW != 0 {
		t.Error("SetLineHeight did not invalidate cell metrics")
	}
	if tm.cfg.LineHeight != 1.2 {
		t.Errorf("cfg.LineHeight = %v, want 1.2", tm.cfg.LineHeight)
	}
}
