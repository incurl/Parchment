package fold

import "math"

// Placement is the similarity that puts a layout authored in a local
// frame onto the sheet.
//
// It is a pair of axes plus an origin rather than a rotation and a
// translation, because the letter is mirrored inside the panel: the two
// sides of a sheet are mirror images of each other, and the letter has
// to read correctly from Side A. The mirrored case cannot be written as
// a rotation.
type Placement struct {
	// Origin is the sheet position of the local frame's (0,0).
	Origin Vec2 `json:"origin"`
	// XAxis is the sheet-space image of the local +x unit vector.
	XAxis Vec2 `json:"x_axis"`
	// YAxis is the sheet-space image of the local +y unit vector.
	YAxis Vec2 `json:"y_axis"`
	// RotationDeg is the angle of the baseline in the sheet frame.
	RotationDeg float64 `json:"rotation_deg"`
	// Mirrored reports that the local frame is reflected.
	Mirrored bool `json:"mirrored"`
}

// Apply maps a point from the local frame onto the sheet.
func (p Placement) Apply(v Vec2) Vec2 {
	return p.Origin.Add(p.XAxis.Mul(v.X)).Add(p.YAxis.Mul(v.Y))
}

// ApplyPolygon maps every vertex of a local-frame polygon onto the
// sheet.
func (p Placement) ApplyPolygon(poly Polygon) Polygon {
	out := make(Polygon, len(poly))
	for i, v := range poly {
		out[i] = p.Apply(v)
	}
	return out
}

// SheetPlacement maps the panel frame onto the sheet, unmirrored. It is
// the placement of everything printed on Side B: the address, the
// return address, the stamp and the fold-plan overlay.
func (s Scene) sheetPlacement() Placement {
	phi := s.FrontPanel.RotationDeg * math.Pi / 180
	return Placement{
		Origin:      s.FrontPanel.TransformToFlat.Apply(Vec2{X: 0, Y: 0}),
		XAxis:       Vec2{X: math.Cos(phi), Y: math.Sin(phi)},
		YAxis:       Vec2{X: -math.Sin(phi), Y: math.Cos(phi)},
		RotationDeg: s.FrontPanel.RotationDeg,
		Mirrored:    false,
	}
}

// LetterPlacement maps the letter's own frame onto the sheet.
//
// # Why the letter is mirrored
//
// Side A and Side B are opposite faces of the same paper, so a design
// authored in one frame appears mirrored when the sheet is turned over.
// The letter is authored with its own left-to-right axis, and the fold
// engine maps that axis onto the panel's -u direction:
//
//	u = W - a,   v = b
//
// The recipient then reads the letter normally when they look at the
// inside of the opened packet, and the text stays inside the front
// panel. The baseline is consequently printed at phi + 180 degrees on
// the sheet, which the PDF exporter mirrors back to -phi on the printed
// Side A page. Both numbers come from the geometry; neither is a
// constant.
func (s Scene) letterPlacement() Placement {
	phi := s.FrontPanel.RotationDeg * math.Pi / 180
	c, sn := math.Cos(phi), math.Sin(phi)
	return Placement{
		Origin:      s.FrontPanel.TransformToFlat.Apply(Vec2{X: s.FrontPanel.Size.Width, Y: 0}),
		XAxis:       Vec2{X: -c, Y: -sn},
		YAxis:       Vec2{X: -sn, Y: c},
		RotationDeg: normalizeDeg(s.FrontPanel.RotationDeg + 180),
		Mirrored:    true,
	}
}

// DerivedRotationDeg recomputes the printed rotation from the front
// panel's corners.
//
// This is the computation the acceptance criteria ask for: the angle is
// the direction of the panel's first edge in the sheet frame, taken from
// the polygon the partition produced. No renderer may use a literal 45
// or 15 degrees; they read this value.
func (s Scene) DerivedRotationDeg() float64 {
	poly := s.FrontPanel.Polygon
	if len(poly) < 2 {
		return 0
	}
	d := poly[1].Sub(poly[0])
	return normalizeDeg(d.AngleRad() * 180 / math.Pi)
}

// AddressRotationDeg returns the angle the address block is printed at
// on the flat sheet relative to the sheet's x axis.
//
// It is DerivedRotationDeg, named for the UI. The value is what the
// inspector displays as "printed at N degrees on Side B".
func (s Scene) AddressRotationDeg() float64 {
	return s.DerivedRotationDeg()
}

// normalizeDeg folds an angle into (-180, 180].
func normalizeDeg(deg float64) float64 {
	for deg <= -180 {
		deg += 360
	}
	for deg > 180 {
		deg -= 360
	}
	return deg
}
