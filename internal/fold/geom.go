// Package fold is the fold-geometry engine of Parchment.
//
// It owns the paper model: where the front panel sits on the A4 sheet,
// how the sheet is partitioned into rigid panels, which panel is hinged
// to which, and in what order the hinges close. Every other part of the
// application - the 2D print preview, the 3D viewer, the PDF exporter -
// consumes the description this package produces instead of computing
// fold geometry of its own.
//
// # The model
//
// The sheet is partitioned into exactly nine rigid panels:
//
//   - front - the visible front panel. It is the root of the hinge tree
//     and never moves.
//   - left, right, top, bottom - the four edge flaps, each hinged on one
//     edge of the front panel.
//   - four corner tucks - the sheet's corners, each hinged on the edge
//     flap that carries it.
//
// The front panel is the root and stays in the world XY plane, so the
// invariant "the front is flat, one layer, and no crease crosses it" is
// structural: it cannot be violated by an animation parameter.
//
// # Frames
//
// Two planar frames are used throughout, both in millimetres:
//
//   - the sheet frame, x right and y down, origin at the sheet's
//     top-left corner, exactly as the page is printed;
//   - the panel frame, u right and v down, origin at the front panel's
//     top-left corner. This is the frame the address reads upright in.
//
// The front panel is rotated by Params.RotationDeg inside the sheet. That
// rotation is what produces the diagonal address: a layout authored in
// the panel frame is printed rotated on Side B, and reads upright on the
// folded packet. See address.go for the derivation.
package fold

import (
	"math"
)

// Epsilon is the geometric tolerance in millimetres.
const Epsilon = 1e-6

// Vec2 is a point or a direction in millimetres.
type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Add returns p+q.
func (p Vec2) Add(q Vec2) Vec2 { return Vec2{p.X + q.X, p.Y + q.Y} }

// Sub returns p-q.
func (p Vec2) Sub(q Vec2) Vec2 { return Vec2{p.X - q.X, p.Y - q.Y} }

// Mul scales p by f.
func (p Vec2) Mul(f float64) Vec2 { return Vec2{p.X * f, p.Y * f} }

// Dot returns the dot product of p and q.
func (p Vec2) Dot(q Vec2) float64 { return p.X*q.X + p.Y*q.Y }

// Length returns the Euclidean length of p.
func (p Vec2) Length() float64 { return math.Hypot(p.X, p.Y) }

// AngleRad returns the angle of p, measured from the +x axis.
func (p Vec2) AngleRad() float64 { return math.Atan2(p.Y, p.X) }

// Rotate rotates p by deg degrees. The sheet frame has y pointing down,
// so a positive angle is clockwise on screen.
func (p Vec2) Rotate(deg float64) Vec2 {
	rad := deg * math.Pi / 180
	c, s := math.Cos(rad), math.Sin(rad)
	return Vec2{p.X*c - p.Y*s, p.X*s + p.Y*c}
}

// Lerp interpolates between p and q.
func Lerp(p, q Vec2, t float64) Vec2 {
	return Vec2{p.X + (q.X-p.X)*t, p.Y + (q.Y-p.Y)*t}
}

// RotateAround rotates p by deg degrees about centre.
func RotateAround(p, centre Vec2, deg float64) Vec2 {
	return centre.Add(p.Sub(centre).Rotate(deg))
}

// Polygon is a closed ring of vertices, in millimetres.
type Polygon []Vec2

// ClipHalfPlane removes the part of the polygon where n.p > c, keeping
// the half-plane n.p <= c.
//
// It is the Sutherland-Hodgman algorithm specialised to one plane, which
// is all the fold partition needs: every panel is the sheet clipped by a
// handful of half-planes.
func ClipHalfPlane(poly Polygon, n Vec2, c float64) Polygon {
	if len(poly) == 0 {
		return nil
	}

	out := make(Polygon, 0, len(poly)+2)
	for i := range poly {
		a := poly[i]
		b := poly[(i+1)%len(poly)]

		da := n.Dot(a) - c
		db := n.Dot(b) - c
		inA := da <= Epsilon
		inB := db <= Epsilon

		if inA {
			out = append(out, a)
		}
		if inA != inB {
			// The edge crosses the plane. da and db have opposite
			// signs, so the denominator cannot vanish.
			t := da / (da - db)
			out = append(out, Lerp(a, b, t))
		}
	}
	return out
}

// Reverse returns the polygon with its vertices in the opposite order.
func (p Polygon) Reverse() Polygon {
	out := make(Polygon, len(p))
	for i := range p {
		out[len(p)-1-i] = p[i]
	}
	return out
}

// Area returns the absolute area of the polygon in square millimetres.
func (p Polygon) Area() float64 {
	var sum float64
	for i := range p {
		a := p[i]
		b := p[(i+1)%len(p)]
		sum += a.X*b.Y - b.X*a.Y
	}
	return math.Abs(sum) / 2
}

// Perimeter returns the closed perimeter of the polygon.
func (p Polygon) Perimeter() float64 {
	var sum float64
	for i := range p {
		sum += p[i].Sub(p[(i+1)%len(p)]).Length()
	}
	return sum
}

// Centroid returns the area centroid of the polygon. It falls back to
// the vertex average for a degenerate polygon.
func (p Polygon) Centroid() Vec2 {
	var cross, cx, cy float64
	for i := range p {
		a := p[i]
		b := p[(i+1)%len(p)]
		k := a.X*b.Y - b.X*a.Y
		cross += k
		cx += (a.X + b.X) * k
		cy += (a.Y + b.Y) * k
	}
	if math.Abs(cross) < Epsilon {
		var sum Vec2
		for _, v := range p {
			sum = sum.Add(v)
		}
		return sum.Mul(1 / float64(len(p)))
	}
	return Vec2{cx / (3 * cross), cy / (3 * cross)}
}

// Bounds returns the axis-aligned bounding box of the polygon.
func (p Polygon) Bounds() (min, max Vec2) {
	if len(p) == 0 {
		return Vec2{}, Vec2{}
	}
	min, max = p[0], p[0]
	for _, v := range p[1:] {
		if v.X < min.X {
			min.X = v.X
		}
		if v.Y < min.Y {
			min.Y = v.Y
		}
		if v.X > max.X {
			max.X = v.X
		}
		if v.Y > max.Y {
			max.Y = v.Y
		}
	}
	return min, max
}

// Contains reports whether the point is inside the polygon, using the
// even-odd rule. A point on an edge counts as inside.
func (p Polygon) Contains(pt Vec2) bool {
	inside := false
	for i, j := 0, len(p)-1; i < len(p); j, i = i, i+1 {
		bi, bj := p[i], p[j]
		if pointOnSegment(bj, bi, pt) {
			return true
		}
		if (bi.Y > pt.Y) != (bj.Y > pt.Y) {
			x := (bj.X-bi.X)*(pt.Y-bi.Y)/(bj.Y-bi.Y) + bi.X
			if pt.X < x {
				inside = !inside
			}
		}
	}
	return inside
}

// pointOnSegment reports whether pt lies on the segment a-b.
func pointOnSegment(a, b, pt Vec2) bool {
	cross := (b.X-a.X)*(pt.Y-a.Y) - (b.Y-a.Y)*(pt.X-a.X)
	if math.Abs(cross) > 1e-6 {
		return false
	}
	return pt.X >= math.Min(a.X, b.X)-1e-6 && pt.X <= math.Max(a.X, b.X)+1e-6 &&
		pt.Y >= math.Min(a.Y, b.Y)-1e-6 && pt.Y <= math.Max(a.Y, b.Y)+1e-6
}

// Rect is an axis-aligned rectangle in the panel frame, in millimetres.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Polygon returns the rectangle's corners in clockwise order.
func (r Rect) Polygon() Polygon {
	return Polygon{
		{X: r.X, Y: r.Y},
		{X: r.X + r.Width, Y: r.Y},
		{X: r.X + r.Width, Y: r.Y + r.Height},
		{X: r.X, Y: r.Y + r.Height},
	}
}

// Contains reports whether the point is inside the rectangle.
func (r Rect) Contains(p Vec2) bool {
	return p.X >= r.X && p.X <= r.X+r.Width && p.Y >= r.Y && p.Y <= r.Y+r.Height
}

// Size is a width and a height in millimetres.
type Size struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Frame is the similarity transform that places the panel frame inside
// the sheet frame: p_sheet = R(rotation) * p_panel + origin.
type Frame struct {
	Origin      Vec2
	RotationDeg float64
}

// PanelToSheet maps a panel-frame point onto the flat sheet.
func (f Frame) PanelToSheet(p Vec2) Vec2 {
	return p.Rotate(f.RotationDeg).Add(f.Origin)
}

// SheetToPanel maps a flat-sheet point into the panel frame.
func (f Frame) SheetToPanel(p Vec2) Vec2 {
	return p.Sub(f.Origin).Rotate(-f.RotationDeg)
}

// Transform is a rotation followed by a translation:
// p' = R(rotation_deg) * p + (tx, ty).
//
// It is the shape the scene description carries for the front panel, so
// the frontend and the PDF exporter can map between the two frames
// without repeating the trigonometry.
type Transform struct {
	RotationDeg float64 `json:"rotation_deg"`
	TX          float64 `json:"tx"`
	TY          float64 `json:"ty"`
}

// Apply maps p through the transform.
func (t Transform) Apply(p Vec2) Vec2 {
	return p.Rotate(t.RotationDeg).Add(Vec2{X: t.TX, Y: t.TY})
}

// Inverse returns the transform that undoes t.
func (t Transform) Inverse() Transform {
	// p' = R p + T  =>  p = R^-1 (p' - T)
	rotInv := Vec2{-t.TX, -t.TY}.Rotate(-t.RotationDeg)
	return Transform{RotationDeg: -t.RotationDeg, TX: rotInv.X, TY: rotInv.Y}
}

// ApplyPolygon maps every vertex through the transform.
func (t Transform) ApplyPolygon(p Polygon) Polygon {
	out := make(Polygon, len(p))
	for i, v := range p {
		out[i] = t.Apply(v)
	}
	return out
}

// Vec3 is a point in the 3D world used by the viewer, in millimetres.
type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Transform3 is a 4x4 affine matrix in row-major order.
//
// The engine uses it to verify the fold invariants: it composes the same
// hinge transforms the viewer composes, so a test can assert where every
// panel ends up at a given fold progress.
type Transform3 [16]float64

// Identity3 returns the identity matrix.
func Identity3() Transform3 {
	return Transform3{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// Mul returns t * u, so that (t*u).Apply(p) == t.Apply(u.Apply(p)).
func (t Transform3) Mul(u Transform3) Transform3 {
	var out Transform3
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			var sum float64
			for k := 0; k < 4; k++ {
				sum += t[r*4+k] * u[k*4+c]
			}
			out[r*4+c] = sum
		}
	}
	return out
}

// Apply transforms p.
func (t Transform3) Apply(p Vec3) Vec3 {
	return Vec3{
		X: t[0]*p.X + t[1]*p.Y + t[2]*p.Z + t[3],
		Y: t[4]*p.X + t[5]*p.Y + t[6]*p.Z + t[7],
		Z: t[8]*p.X + t[9]*p.Y + t[10]*p.Z + t[11],
	}
}

// Translation3 returns a translation matrix.
func Translation3(d Vec3) Transform3 {
	m := Identity3()
	m[3], m[7], m[11] = d.X, d.Y, d.Z
	return m
}

// Rotation3 returns the rotation of angleDeg about the line through
// origin with unit direction axis (Rodrigues' formula).
func Rotation3(origin Vec3, axis Vec3, angleDeg float64) Transform3 {
	rad := angleDeg * math.Pi / 180
	c, s := math.Cos(rad), math.Sin(rad)
	u := axis
	// Normalise defensively: the axis is built from two points.
	if n := math.Sqrt(u.X*u.X + u.Y*u.Y + u.Z*u.Z); n > 0 {
		u = Vec3{u.X / n, u.Y / n, u.Z / n}
	}

	// Rotation about the origin, then shift so the axis passes
	// through origin.
	m := Transform3{
		c + u.X*u.X*(1-c), u.X*u.Y*(1-c) - u.Z*s, u.X*u.Z*(1-c) + u.Y*s, 0,
		u.Y*u.X*(1-c) + u.Z*s, c + u.Y*u.Y*(1-c), u.Y*u.Z*(1-c) - u.X*s, 0,
		u.Z*u.X*(1-c) - u.Y*s, u.Z*u.Y*(1-c) + u.X*s, c + u.Z*u.Z*(1-c), 0,
		0, 0, 0, 1,
	}
	return Translation3(origin).Mul(m).Mul(Translation3(Vec3{-origin.X, -origin.Y, -origin.Z}))
}
