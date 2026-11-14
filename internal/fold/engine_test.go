package fold

import (
	"math"
	"testing"
)

// cases returns a spread of parameter sets that must all fold.
func cases() []Params {
	base := DefaultParams()
	out := []Params{base}
	for _, phi := range []float64{-20, -10, 0, 8, 15, 25, 35} {
		p := base
		p.RotationDeg = phi
		out = append(out, p)
	}
	for _, size := range [][2]float64{{120, 90}, {150, 115}, {170, 105}, {140, 120}} {
		for _, phi := range []float64{0, 12, 22} {
			p := base
			p.PanelWidth, p.PanelHeight, p.RotationDeg = size[0], size[1], phi
			out = append(out, p)
		}
	}
	p := base
	p.OffsetX, p.OffsetY = 8, -12
	out = append(out, p)
	p = base
	p.TuckDepth = 2
	out = append(out, p)
	return out
}

// mustBuild builds a scene or fails the test.
func mustBuild(t *testing.T, p Params) Scene {
	t.Helper()
	scene, err := Build(p)
	if err != nil {
		t.Fatalf("Build(%+v): %v", p, err)
	}
	return scene
}

// TestPartitionTilesTheSheet is the geometric soundness test: the panel
// areas must add up to the sheet, every panel must have a parent that
// exists, and exactly one panel is the root.
func TestPartitionTilesTheSheet(t *testing.T) {
	for _, p := range cases() {
		scene := mustBuild(t, p)

		var sum float64
		ids := make(map[string]Panel, len(scene.Panels))
		roots := 0
		for _, panel := range scene.Panels {
			sum += panel.Polygon.Area()
			if len(panel.Polygon) < 3 {
				t.Errorf("%+v: panel %s has %d vertices", p, panel.ID, len(panel.Polygon))
			}
			if panel.Parent == "" {
				roots++
				if panel.ID != PanelFront {
					t.Errorf("%+v: %s is a root", p, panel.ID)
				}
			}
			ids[panel.ID] = panel
		}

		want := p.SheetWidth * p.SheetHeight
		if math.Abs(sum-want) > 0.5 {
			t.Errorf("%+v: panels cover %.2f mm2, sheet is %.2f mm2", p, sum, want)
		}
		if roots != 1 {
			t.Errorf("%+v: %d roots, want 1", p, roots)
		}
		for _, panel := range scene.Panels {
			if panel.Parent != "" {
				if _, ok := ids[panel.Parent]; !ok {
					t.Errorf("%+v: %s has missing parent %s", p, panel.ID, panel.Parent)
				}
			}
		}

		// Every folding panel must be in the schedule, and nothing else.
		if len(scene.FoldOrder) != len(scene.Panels)-1 {
			t.Errorf("%+v: fold order has %d entries for %d folding panels",
				p, len(scene.FoldOrder), len(scene.Panels)-1)
		}
		for _, id := range scene.FoldOrder {
			if _, ok := ids[id]; !ok {
				t.Errorf("%+v: fold order names missing panel %s", p, id)
			}
		}
	}
}

// TestPartitionHasNoOverlaps samples the flat sheet and asserts that
// every point belongs to exactly one panel.
func TestPartitionHasNoOverlaps(t *testing.T) {
	for _, p := range cases() {
		scene := mustBuild(t, p)

		// A jittered grid avoids landing exactly on a shared edge,
		// where two panels legitimately both contain the point.
		for y := 3.7; y < p.SheetHeight-1; y += 7.3 {
			for x := 2.9; x < p.SheetWidth-1; x += 6.9 {
				pt := Vec2{X: x, Y: y}

				onBoundary := false
				for _, panel := range scene.Panels {
					if onEdge(panel.SheetPolygon, pt, 0.05) {
						onBoundary = true
						break
					}
				}
				if onBoundary {
					continue
				}

				hits := 0
				for _, panel := range scene.Panels {
					if panel.SheetPolygon.Contains(pt) {
						hits++
					}
				}
				if hits != 1 {
					t.Fatalf("%+v: sheet point (%.1f, %.1f) is in %d panels", p, x, y, hits)
				}
			}
		}
	}
}

// onEdge reports whether pt lies within tol of one of the polygon's
// edges.
func onEdge(poly Polygon, pt Vec2, tol float64) bool {
	for i := range poly {
		a := poly[i]
		b := poly[(i+1)%len(poly)]
		d := b.Sub(a)
		length2 := d.Dot(d)
		if length2 < 1e-12 {
			continue
		}
		t := math.Max(0, math.Min(1, pt.Sub(a).Dot(d)/length2))
		if pt.Sub(a.Add(d.Mul(t))).Length() <= tol {
			return true
		}
	}
	return false
}
