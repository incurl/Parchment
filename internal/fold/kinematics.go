package fold

import "math"

// AngleAt returns the hinge's angle at fold progress t, in degrees.
//
// This is the only per-frame computation the fold needs: the hinge
// angle is a linear interpolation over the panel's window, clamped to
// the window. The viewer calls the same function on the scene it was
// given; it never recomputes geometry.
func (p Panel) AngleAt(t float64) float64 {
	if p.Hinge == nil || p.TEnd <= p.TStart {
		return 0
	}
	progress := (t - p.TStart) / (p.TEnd - p.TStart)
	progress = math.Max(0, math.Min(1, progress))
	return p.Hinge.AngleDeg * progress
}

// TransformsAt returns the world transform of every panel at fold
// progress t.
//
// The engine uses it to verify the fold invariants, and the 3D viewer
// performs the same composition in JavaScript. Panels are resolved
// parent-first, so a child's matrix already contains every ancestor's
// rotation.
func (s Scene) TransformsAt(t float64) map[string]Transform3 {
	byID := make(map[string]Panel, len(s.Panels))
	for _, panel := range s.Panels {
		byID[panel.ID] = panel
	}

	out := make(map[string]Transform3, len(s.Panels))
	var resolve func(panel Panel) Transform3
	resolve = func(panel Panel) Transform3 {
		if m, ok := out[panel.ID]; ok {
			return m
		}

		parent := Identity3()
		if panel.Parent != "" {
			if p, ok := byID[panel.Parent]; ok {
				parent = resolve(p)
			}
		}

		local := Identity3()
		if panel.Hinge != nil {
			angle := panel.AngleAt(t)
			if math.Abs(angle) > Epsilon {
				local = Rotation3(panel.Hinge.A, panel.Hinge.Axis, angle)
			}
		}

		m := parent.Mul(local)
		out[panel.ID] = m
		return m
	}

	for _, panel := range s.Panels {
		resolve(panel)
	}
	return out
}

// FoldedPolygon returns where a panel's vertices are at fold progress t.
func (s Scene) FoldedPolygon(panel Panel, t float64) Polygon {
	m := s.TransformsAt(t)[panel.ID]
	out := make(Polygon, len(panel.Polygon))
	for i, v := range panel.Polygon {
		p := m.Apply(Vec3{X: v.X, Y: v.Y})
		out[i] = Vec2{X: p.X, Y: p.Y}
	}
	return out
}

// FoldedNormal returns the panel's unit normal at fold progress t.
//
// A fresh panel faces +Z. The normal flips once per hinge in its chain,
// so the front panel always faces +Z and every panel behind it faces
// -Z after its fold completes.
func (s Scene) FoldedNormal(panel Panel, t float64) Vec3 {
	m := s.TransformsAt(t)[panel.ID]
	// The third column of the 3x3 part is the image of +Z.
	return normalize3(Vec3{X: m[2], Y: m[6], Z: m[10]})
}

// normalize3 returns the unit vector in the direction of v.
func normalize3(v Vec3) Vec3 {
	n := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if n < Epsilon {
		return Vec3{}
	}
	return Vec3{X: v.X / n, Y: v.Y / n, Z: v.Z / n}
}
