package fold

import (
	"math"
	"testing"
)

// TestAddressTransformRoundTrip is the acceptance test for the diagonal
// address: applying the fold's own transforms must reproduce the printed
// rectangle exactly.
func TestAddressTransformRoundTrip(t *testing.T) {
	for _, p := range cases() {
		scene := mustBuild(t, p)
		toFlat := scene.FrontPanel.TransformToFlat
		fromFlat := scene.FrontPanel.TransformFromFlat

		// The two transforms must be exact inverses.
		for _, pt := range scene.Layout.AddressRect.Polygon() {
			back := fromFlat.Apply(toFlat.Apply(pt))
			if back.Sub(pt).Length() > 1e-9 {
				t.Fatalf("%+v: round trip moved (%.2f, %.2f) to (%.2f, %.2f)",
					p, pt.X, pt.Y, back.X, back.Y)
			}
		}

		// And so must the composition of the placement and the
		// inverse: a printed point maps back onto the layout.
		placement := scene.SheetPlacement
		for _, pt := range scene.Layout.ReturnRect.Polygon() {
			printed := placement.Apply(pt)
			back := fromFlat.Apply(printed)
			if back.Sub(pt).Length() > 1e-9 {
				t.Fatalf("%+v: placement round trip moved (%.2f, %.2f)", p, pt.X, pt.Y)
			}
		}
	}
}

// TestPrintedRotationIsDerivedFromGeometry checks that the printed angle
// comes from the front panel's corners rather than from a constant.
func TestPrintedRotationIsDerivedFromGeometry(t *testing.T) {
	for _, p := range cases() {
		scene := mustBuild(t, p)

		derived := scene.DerivedRotationDeg()
		want := normalizeDeg(p.RotationDeg)
		if math.Abs(derived-want) > 1e-6 {
			t.Errorf("%+v: derived rotation %.6f, want %.6f", p, derived, want)
		}
		if math.Abs(scene.AddressRotationDeg()-derived) > 1e-9 {
			t.Errorf("%+v: address rotation %.6f differs from the derived %.6f",
				p, scene.AddressRotationDeg(), derived)
		}
		if math.Abs(scene.FrontPanel.TransformToFlat.RotationDeg-want) > 1e-9 {
			t.Errorf("%+v: transform rotation %.6f, want %.6f",
				p, scene.FrontPanel.TransformToFlat.RotationDeg, want)
		}
	}
}

// TestPrintedAddressIsRotatedButUndistorted checks that the address is
// printed at the derived angle, with its proportions intact.
func TestPrintedAddressIsRotatedButUndistorted(t *testing.T) {
	for _, p := range cases() {
		scene := mustBuild(t, p)
		placement := scene.SheetPlacement
		rect := scene.Layout.AddressRect.Polygon()
		printed := placement.ApplyPolygon(rect)

		// The baseline direction is the first edge's direction.
		edge := printed[1].Sub(printed[0])
		angle := normalizeDeg(edge.AngleRad() * 180 / math.Pi)
		if math.Abs(angle-scene.DerivedRotationDeg()) > 1e-6 {
			t.Errorf("%+v: printed baseline is at %.4f degrees, want %.4f",
				p, angle, scene.DerivedRotationDeg())
		}

		for i := range rect {
			want := rect[(i+1)%len(rect)].Sub(rect[i]).Length()
			got := printed[(i+1)%len(printed)].Sub(printed[i]).Length()
			if math.Abs(got-want) > 1e-9 {
				t.Errorf("%+v: printed edge %d is %.4f mm, want %.4f mm", p, i, got, want)
			}
		}

		// The printed block must land on the front panel's paper.
		for _, v := range printed {
			if !scene.FrontPanel.Polygon.Contains(v) {
				t.Errorf("%+v: printed address corner (%.2f, %.2f) is off the front panel",
					p, v.X, v.Y)
			}
		}
	}
}

// TestLetterPlacementReadsFromSideA checks the mirrored letter transform:
// it mirrors the panel's u axis, keeps the text inside the panel, and
// prints at the derived angle plus a half turn.
func TestLetterPlacementReadsFromSideA(t *testing.T) {
	for _, p := range cases() {
		scene := mustBuild(t, p)
		placement := scene.LetterPlacement

		if !placement.Mirrored {
			t.Errorf("%+v: the letter placement is not mirrored", p)
		}

		// The local +x axis maps onto the panel's -u direction.
		panelX := scene.SheetPlacement.XAxis
		if placement.XAxis.Add(panelX).Length() > 1e-9 {
			t.Errorf("%+v: letter x axis %+v is not the panel's -u axis %+v",
				p, placement.XAxis, panelX.Mul(-1))
		}

		want := normalizeDeg(p.RotationDeg + 180)
		if math.Abs(placement.RotationDeg-want) > 1e-9 {
			t.Errorf("%+v: letter rotation %.3f, want %.3f", p, placement.RotationDeg, want)
		}

		// The letter's safe area must be inside the front panel.
		for _, v := range placement.ApplyPolygon(scene.Layout.LetterRect.Polygon()) {
			if !scene.FrontPanel.Polygon.Contains(v) {
				t.Errorf("%+v: letter corner (%.2f, %.2f) leaves the front panel", p, v.X, v.Y)
			}
		}
	}
}

// TestRotationFollowsTheParameters is the "changing fold angles keeps the
// address upright" criterion: the transform tracks each parameter change
// and the printed rectangle stays inside the panel.
func TestRotationFollowsTheParameters(t *testing.T) {
	for _, phi := range []float64{-25, -8, 0, 6, 14, 27, 40} {
		p := DefaultParams()
		p.RotationDeg = phi
		scene, err := Build(p)
		if err != nil {
			continue // some angles do not fit; that is rejected, not misprinted
		}

		want := normalizeDeg(phi)
		if got := scene.DerivedRotationDeg(); math.Abs(got-want) > 1e-6 {
			t.Fatalf("phi=%.1f: derived %.4f", phi, got)
		}
		for _, v := range scene.SheetPlacement.ApplyPolygon(scene.Layout.AddressRect.Polygon()) {
			if !scene.FrontPanel.Polygon.Contains(v) {
				t.Fatalf("phi=%.1f: address corner off the panel", phi)
			}
		}
	}
}

// TestStoredTransformIsUsableByTheFrontend checks the JSON shape the
// frontend receives: a rotation and a translation, no matrix.
func TestStoredTransformIsUsableByTheFrontend(t *testing.T) {
	scene := mustBuild(t, DefaultParams())
	tf := scene.FrontPanel.TransformToFlat
	if tf.RotationDeg == 0 {
		t.Fatal("the default fold should print the address at a non-zero angle")
	}
	if tf.TX == 0 && tf.TY == 0 {
		t.Fatal("the transform lost its translation")
	}
	// A point at the panel's origin maps onto the panel's first corner.
	origin := tf.Apply(Vec2{})
	if origin.Sub(scene.FrontPanel.Polygon[0]).Length() > 1e-9 {
		t.Fatalf("panel origin maps to %+v, want %+v", origin, scene.FrontPanel.Polygon[0])
	}
}
