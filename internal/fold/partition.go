package fold

import (
	"fmt"
	"math"
)

// makeFrame places the front panel inside the sheet.
//
// The panel's centre is the sheet's centre plus the offset, so the
// default fold is symmetric and the offset biases the flaps.
func makeFrame(p Params) Frame {
	phi := p.RotationDeg * math.Pi / 180
	c, s := math.Cos(phi), math.Sin(phi)
	halfU, halfV := p.PanelWidth/2, p.PanelHeight/2
	centreSheet := Vec2{
		X: p.SheetWidth/2 + p.OffsetX,
		Y: p.SheetHeight/2 + p.OffsetY,
	}
	// O = centreSheet - R(phi) * (halfU, halfV)
	origin := Vec2{
		X: centreSheet.X - (c*halfU - s*halfV),
		Y: centreSheet.Y - (s*halfU + c*halfV),
	}
	return Frame{Origin: origin, RotationDeg: p.RotationDeg}
}

// sheetInPanelFrame returns the sheet's four corners in the panel frame.
func sheetInPanelFrame(p Params, f Frame) Polygon {
	sheet := Polygon{
		{X: 0, Y: 0},
		{X: p.SheetWidth, Y: 0},
		{X: p.SheetWidth, Y: p.SheetHeight},
		{X: 0, Y: p.SheetHeight},
	}
	out := make(Polygon, len(sheet))
	for i, v := range sheet {
		out[i] = f.SheetToPanel(v)
	}
	return out
}

// mapFrame maps a panel-frame polygon onto the sheet.
func mapFrame(poly Polygon, f Frame) Polygon {
	out := make(Polygon, len(poly))
	for i, v := range poly {
		out[i] = f.PanelToSheet(v)
	}
	return out
}

// toWorld maps a panel-frame polygon into the viewer's world frame.
//
// The world frame puts the front panel's centre at the origin, x to the
// right and y up, so the camera and the lights do not need to know about
// the sheet's top-left origin.
func toWorld(poly Polygon, p Params) Polygon {
	out := make(Polygon, len(poly))
	for i, v := range poly {
		out[i] = Vec2{X: v.X - p.PanelWidth/2, Y: p.PanelHeight/2 - v.Y}
	}
	return out
}

// worldPoint maps a single panel-frame point into the world frame.
func worldPoint(v Vec2, p Params) Vec3 {
	return Vec3{X: v.X - p.PanelWidth/2, Y: p.PanelHeight/2 - v.Y}
}

// frameToTransform expresses a frame as a Transform.
func frameToTransform(f Frame) Transform {
	return Transform{RotationDeg: f.RotationDeg, TX: f.Origin.X, TY: f.Origin.Y}
}

// insidePolygon reports whether every vertex of inner is inside outer
// and the centroids also agree. It is used to reject a panel that the
// rotation pushed off the sheet.
func insidePolygon(inner, outer Polygon) bool {
	if len(inner) == 0 || len(outer) == 0 {
		return false
	}
	if !outer.Contains(inner.Centroid()) {
		return false
	}
	for _, v := range inner {
		if !outer.Contains(v) {
			return false
		}
	}
	return true
}

// atMost clips the polygon to coordinate <= c on the given axis.
func atMost(p Polygon, axis int, c float64) Polygon {
	if axis == 0 {
		return ClipHalfPlane(p, Vec2{X: 1}, c)
	}
	return ClipHalfPlane(p, Vec2{Y: 1}, c)
}

// atLeast clips the polygon to coordinate >= c on the given axis.
func atLeast(p Polygon, axis int, c float64) Polygon {
	if axis == 0 {
		return ClipHalfPlane(p, Vec2{X: -1}, -c)
	}
	return ClipHalfPlane(p, Vec2{Y: -1}, -c)
}

// between clips the polygon to c0 <= coordinate <= c1 on the given axis.
func between(p Polygon, axis int, c0, c1 float64) Polygon {
	return atMost(atLeast(p, axis, c0), axis, c1)
}

// ccw returns the polygon wound counter-clockwise in the world frame, so
// every panel's normal points at +Z before it folds.
func ccw(p Polygon) Polygon {
	var signed float64
	for i := range p {
		a := p[i]
		b := p[(i+1)%len(p)]
		signed += a.X*b.Y - b.X*a.Y
	}
	if signed < 0 {
		return p.Reverse()
	}
	return p
}

// partition cuts the sheet into the nine rigid panels.
//
// The cuts are the four lines through the front panel's edges plus the
// four corner creases parallel to them. In the panel frame:
//
//	front   [0,W] x [0,H]
//	left    u <= 0,  -k <= v <= H+k
//	right   u >= W,  -k <= v <= H+k
//	top     0 <= u <= W, v <= 0
//	bottom  0 <= u <= W, v >= H
//	tuck-tl u <= 0, v <= -k      (and the three rotations)
//
// Every region is the sheet clipped by a few half-planes, so the pieces
// are exact and their areas add up to the sheet's area - the test in
// partition_test.go asserts that.
func partition(p Params, frame Frame, sheet Polygon) ([]Panel, error) {
	k := p.TuckDepth
	width, height := p.PanelWidth, p.PanelHeight

	type spec struct {
		id     string
		parent string
		role   string
		poly   Polygon
		layer  int
	}

	specs := []spec{
		{
			id: PanelFront, parent: "", role: "Front panel (letter side in, address side out)",
			poly:  Rect{X: 0, Y: 0, Width: width, Height: height}.Polygon(),
			layer: 0,
		},
		{
			id: PanelLeft, parent: PanelFront, role: "Left side flap",
			poly:  between(atMost(sheet, 0, 0), 1, -k, height+k),
			layer: 2,
		},
		{
			id: PanelRight, parent: PanelFront, role: "Right side flap",
			poly:  between(atLeast(sheet, 0, width), 1, -k, height+k),
			layer: 2,
		},
		{
			id: PanelBottom, parent: PanelFront, role: "Bottom flap",
			poly:  atLeast(between(sheet, 0, 0, width), 1, height),
			layer: 3,
		},
		{
			id: PanelTop, parent: PanelFront, role: "Final flap (sealed last)",
			poly:  atMost(between(sheet, 0, 0, width), 1, 0),
			layer: 4,
		},
		{
			id: TuckTopLeft, parent: PanelLeft, role: "Diagonal corner fold",
			poly:  atMost(atMost(sheet, 0, 0), 1, -k),
			layer: 1,
		},
		{
			id: TuckTopRight, parent: PanelRight, role: "Diagonal corner fold",
			poly:  atMost(atLeast(sheet, 0, width), 1, -k),
			layer: 1,
		},
		{
			id: TuckBottomLeft, parent: PanelLeft, role: "Diagonal corner fold",
			poly:  atLeast(atMost(sheet, 0, 0), 1, height+k),
			layer: 1,
		},
		{
			id: TuckBottomRight, parent: PanelRight, role: "Diagonal corner fold",
			poly:  atLeast(atLeast(sheet, 0, width), 1, height+k),
			layer: 1,
		},
	}

	hinges := map[string]Hinge{
		PanelLeft:   {A: worldPoint(Vec2{X: 0, Y: 0}, p), B: worldPoint(Vec2{X: 0, Y: height}, p)},
		PanelRight:  {A: worldPoint(Vec2{X: width, Y: 0}, p), B: worldPoint(Vec2{X: width, Y: height}, p)},
		PanelTop:    {A: worldPoint(Vec2{X: 0, Y: 0}, p), B: worldPoint(Vec2{X: width, Y: 0}, p)},
		PanelBottom: {A: worldPoint(Vec2{X: 0, Y: height}, p), B: worldPoint(Vec2{X: width, Y: height}, p)},
	}

	panels := make([]Panel, 0, len(specs))
	for _, s := range specs {
		if len(s.poly) < 3 || s.poly.Area() < 0.5 {
			// A corner the sheet does not reach is simply absent.
			// The area check also drops the slivers a clipping
			// library can leave behind.
			continue
		}

		panel := Panel{
			ID:           s.id,
			Parent:       s.parent,
			Role:         s.role,
			Polygon:      ccw(toWorld(s.poly, p)),
			SheetPolygon: mapFrame(s.poly, frame),
			UV:           textureCoordinates(s.poly, p),
			Layer:        s.layer,
			Tint:         tintFor(s.id),
		}

		if s.id == PanelFront {
			panel.Hinge = nil
			panels = append(panels, panel)
			continue
		}

		hinge, ok := hinges[s.id]
		if !ok {
			hinge, ok = cornerHinge(s.poly, p, s.id)
			if !ok {
				return nil, fmt.Errorf("fold: the corner crease of %s is missing", s.id)
			}
		}
		hinge.Axis = unitDirection(hinge.A, hinge.B)
		hinge.AngleDeg = halfTurnSign(hinge, panel.Polygon.Centroid()) * 180
		window := foldSchedule[s.id]
		panel.Hinge = &hinge
		panel.TStart = window[0]
		panel.TEnd = window[1]
		panels = append(panels, panel)
	}

	if len(panels) < 5 {
		return nil, fmt.Errorf("fold: the partition produced only %d panels", len(panels))
	}
	return panels, nil
}

// textureCoordinates returns the UV of every vertex of a panel.
//
// u is the sheet's x, v is the sheet's y flipped, because an uploaded
// texture has its origin at the bottom-left.
func textureCoordinates(poly Polygon, p Params) Polygon {
	out := make(Polygon, len(poly))
	for i, v := range poly {
		out[i] = Vec2{X: v.X / p.SheetWidth, Y: 1 - v.Y/p.SheetHeight}
	}
	return out
}

// tintFor marks the panels that belong to the back of the packet.
func tintFor(id string) string {
	if id == PanelFront {
		return ""
	}
	return "seam"
}

// cornerHinge returns the crease of a corner tuck.
//
// The crease is parallel to the panel's edge and TuckDepth away from it.
// The segment is the part of that line the tuck actually occupies, so
// the hinge never reaches outside the paper.
func cornerHinge(poly Polygon, p Params, id string) (Hinge, bool) {
	var v float64
	switch id {
	case TuckTopLeft, TuckTopRight:
		v = -p.TuckDepth
	case TuckBottomLeft, TuckBottomRight:
		v = p.PanelHeight + p.TuckDepth
	default:
		return Hinge{}, false
	}

	u0, u1, ok := lineExtent(poly, 1, v)
	if !ok {
		return Hinge{}, false
	}
	return Hinge{
		A: worldPoint(Vec2{X: u0, Y: v}, p),
		B: worldPoint(Vec2{X: u1, Y: v}, p),
	}, true
}

// lineExtent returns the smallest and largest coordinate along the other
// axis where the polygon crosses the line coordinate[axis] == value.
func lineExtent(poly Polygon, axis int, value float64) (float64, float64, bool) {
	other := func(v Vec2) float64 {
		if axis == 0 {
			return v.Y
		}
		return v.X
	}
	this := func(v Vec2) float64 {
		if axis == 0 {
			return v.X
		}
		return v.Y
	}

	min, max := math.Inf(1), math.Inf(-1)
	found := false
	record := func(o float64) {
		min, max, found = math.Min(min, o), math.Max(max, o), true
	}
	for i := range poly {
		a := poly[i]
		b := poly[(i+1)%len(poly)]
		da, db := this(a)-value, this(b)-value
		if math.Abs(da) <= Epsilon {
			record(other(a))
		}
		if (da < 0) != (db < 0) && math.Abs(da-db) > Epsilon {
			t := da / (da - db)
			record(other(a) + (other(b)-other(a))*t)
		}
	}
	return min, max, found
}

// unitDirection returns the unit vector from a to b.
func unitDirection(a, b Vec3) Vec3 {
	d := Vec3{X: b.X - a.X, Y: b.Y - a.Y, Z: b.Z - a.Z}
	n := math.Sqrt(d.X*d.X + d.Y*d.Y + d.Z*d.Z)
	if n < Epsilon {
		return Vec3{}
	}
	return Vec3{X: d.X / n, Y: d.Y / n, Z: d.Z / n}
}

// halfTurnSign returns +1 or -1 so that rotating the centroid about the
// hinge by the signed half-turn moves it towards -Z halfway through.
func halfTurnSign(h Hinge, centroid Vec2) float64 {
	v := Vec3{X: centroid.X - h.A.X, Y: centroid.Y - h.A.Y}
	cross := h.Axis.X*v.Y - h.Axis.Y*v.X
	if cross < 0 {
		return 1
	}
	return -1
}

// foldOrder lists the panels in the order their hinges close.
//
// The order is the fold sequence, not the array order: the viewer uses it
// for the step labels and the tests use it to fold in the right sequence.
func foldOrder(panels []Panel) []string {
	byID := make(map[string]Panel, len(panels))
	for _, p := range panels {
		byID[p.ID] = p
	}

	sequence := []string{
		TuckTopLeft, TuckBottomRight, TuckTopRight, TuckBottomLeft,
		PanelLeft, PanelRight, PanelBottom, PanelTop,
	}
	order := make([]string, 0, len(sequence))
	for _, id := range sequence {
		if _, ok := byID[id]; ok {
			order = append(order, id)
		}
	}
	return order
}
