package fold

import (
	"math"
	"strings"
	"testing"
)

// TestFrontPanelIsInvariant is the acceptance criterion made structural:
// the front panel's transform is the identity at every fold progress, its
// normal stays +Z, and no hinge crosses its interior.
func TestFrontPanelIsInvariant(t *testing.T) {
	for _, p := range cases() {
		scene := mustBuild(t, p)

		var front Panel
		for _, panel := range scene.Panels {
			if panel.ID == PanelFront {
				front = panel
			}
		}
		if front.ID == "" {
			t.Fatalf("%+v: no front panel", p)
		}

		for _, progress := range []float64{0, 0.1, 0.35, 0.6, 0.9, 1} {
			m := scene.TransformsAt(progress)[PanelFront]
			for i, want := range Identity3() {
				if math.Abs(m[i]-want) > 1e-9 {
					t.Fatalf("%+v: front transform at t=%.2f is %v", p, progress, m)
				}
			}
			normal := scene.FoldedNormal(front, progress)
			if math.Abs(normal.Z-1) > 1e-9 || math.Abs(normal.X) > 1e-9 || math.Abs(normal.Y) > 1e-9 {
				t.Fatalf("%+v: front normal at t=%.2f is %+v", p, progress, normal)
			}
		}

		// No crease may pass through the front panel's interior. The
		// packing hinges lie on its boundary, so the test uses an
		// inset rectangle.
		inset := Rect{
			X:      -p.PanelWidth/2 + 0.01,
			Y:      -p.PanelHeight/2 + 0.01,
			Width:  p.PanelWidth - 0.02,
			Height: p.PanelHeight - 0.02,
		}
		for _, panel := range scene.Panels {
			if panel.Hinge == nil {
				continue
			}
			for i := 0; i <= 20; i++ {
				f := float64(i) / 20
				pt := Vec2{
					X: panel.Hinge.A.X + (panel.Hinge.B.X-panel.Hinge.A.X)*f,
					Y: panel.Hinge.A.Y + (panel.Hinge.B.Y-panel.Hinge.A.Y)*f,
				}
				if inset.Contains(pt) {
					t.Fatalf("%+v: hinge of %s crosses the front panel at (%.2f, %.2f)",
						p, panel.ID, pt.X, pt.Y)
				}
			}
		}
	}
}

// TestFoldedPacketIsFlatAndSealed checks the t = 1 state: every flap is
// behind the front (its normal flipped, or flipped twice for the corner
// tucks that are carried by a flap), and no panel reaches far outside the
// packet.
func TestFoldedPacketIsFlatAndSealed(t *testing.T) {
	for _, p := range cases() {
		scene := mustBuild(t, p)
		if scene.Metrics.BackCoverage < 0.995 {
			continue
		}

		for _, panel := range scene.Panels {
			if panel.ID == PanelFront {
				continue
			}

			wantZ := -1.0
			if strings.HasPrefix(panel.ID, "tuck") {
				// A tuck folds once and is then carried by the
				// flap it is hinged to: two half-turns.
				wantZ = 1
			}
			normal := scene.FoldedNormal(panel, 1)
			if math.Abs(normal.Z-wantZ) > 1e-9 {
				t.Errorf("%+v: %s normal at t=1 is %+v, want z=%0.0f",
					p, panel.ID, normal, wantZ)
			}

			// A fold whose flaps are deeper than the panel reaches
			// past its edge. That is a design warning, not a
			// broken hinge, so containment is asserted only for a
			// fold the metrics call clean.
			if scene.Metrics.ProtrusionMM > p.TuckDepth+0.6 {
				continue
			}
			for _, v := range scene.FoldedPolygon(panel, 1) {
				if math.Abs(v.Y) > p.PanelHeight/2+p.TuckDepth+0.6 ||
					math.Abs(v.X) > p.PanelWidth/2+p.TuckDepth+0.6 {
					t.Errorf("%+v: %s reaches (%.1f, %.1f) outside the packet",
						p, panel.ID, v.X, v.Y)
					break
				}
			}
		}
	}
}

// TestAngleAtInterpolates checks the only per-frame computation.
func TestAngleAtInterpolates(t *testing.T) {
	scene := mustBuild(t, DefaultParams())
	for _, panel := range scene.Panels {
		if panel.Hinge == nil {
			if panel.AngleAt(0.5) != 0 {
				t.Errorf("%s has no hinge but returns an angle", panel.ID)
			}
			continue
		}

		if got := panel.AngleAt(panel.TStart); math.Abs(got) > 1e-9 {
			t.Errorf("%s: angle at start = %.3f, want 0", panel.ID, got)
		}
		if got := panel.AngleAt(panel.TEnd); math.Abs(got-panel.Hinge.AngleDeg) > 1e-9 {
			t.Errorf("%s: angle at end = %.3f, want %.3f", panel.ID, got, panel.Hinge.AngleDeg)
		}
		mid := (panel.TStart + panel.TEnd) / 2
		if got := panel.AngleAt(mid); math.Abs(got-panel.Hinge.AngleDeg/2) > 1e-9 {
			t.Errorf("%s: angle at mid = %.3f, want %.3f", panel.ID, got, panel.Hinge.AngleDeg/2)
		}
		if got := panel.AngleAt(-1); math.Abs(got) > 1e-9 {
			t.Errorf("%s: angle before start = %.3f, want 0", panel.ID, got)
		}
		if got := panel.AngleAt(2); math.Abs(got-panel.Hinge.AngleDeg) > 1e-9 {
			t.Errorf("%s: angle after end = %.3f, want %.3f", panel.ID, got, panel.Hinge.AngleDeg)
		}
	}
}

// TestDefaultFoldIsSealed pins the shipped fold: full coverage, no
// protrusion beyond the tuck lip, inside the postal limits and free of
// warnings.
func TestDefaultFoldIsSealed(t *testing.T) {
	p := DefaultParams()
	scene := mustBuild(t, p)
	m := scene.Metrics

	if m.BackCoverage < 0.999 {
		t.Errorf("back coverage is %.3f, want 1", m.BackCoverage)
	}
	if m.ProtrusionMM > p.TuckDepth+0.6 {
		t.Errorf("protrusion is %.2f mm, want at most %.2f", m.ProtrusionMM, p.TuckDepth+0.6)
	}
	if !m.WithinLimits {
		t.Errorf("packet %.2f x %.2f in is outside the limits", m.HeightIn, m.LengthIn)
	}
	if len(m.Warnings) != 0 {
		t.Errorf("default fold has warnings: %v", m.Warnings)
	}
	if len(scene.Panels) != 9 {
		t.Errorf("default fold has %d panels, want 9", len(scene.Panels))
	}
	if len(scene.FoldOrder) != 8 {
		t.Errorf("default fold has %d hinges, want 8", len(scene.FoldOrder))
	}
}

// TestOutOfLimitPacketWarns checks that the postal validation is wired to
// the metrics rather than left as a helper nobody calls.
func TestOutOfLimitPacketWarns(t *testing.T) {
	p := DefaultParams()
	p.PanelWidth, p.PanelHeight = 60, 40
	scene := mustBuild(t, p)
	if scene.Metrics.WithinLimits {
		t.Fatal("a 60 x 40 mm packet cannot be within the USPS letter limits")
	}
	if len(scene.Metrics.Warnings) == 0 {
		t.Fatal("no warning for an out-of-limit packet")
	}
}

// TestBuildRejectsImpossibleFolds checks the guards.
func TestBuildRejectsImpossibleFolds(t *testing.T) {
	p := DefaultParams()
	p.PanelWidth = 400
	if _, err := Build(p); err == nil {
		t.Error("a panel wider than the sheet was accepted")
	}

	p = DefaultParams()
	p.RotationDeg = 90
	if _, err := Build(p); err == nil {
		t.Error("a 90 degree rotation was accepted")
	}

	p = DefaultParams()
	p.PanelWidth, p.PanelHeight, p.RotationDeg = 200, 200, 40
	if _, err := Build(p); err == nil {
		t.Error("a panel that does not fit after rotation was accepted")
	}
}
