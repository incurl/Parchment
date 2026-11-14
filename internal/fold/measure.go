package fold

import (
	"fmt"
	"math"
)

// measure computes the fold's properties: the packet size, whether the
// flaps seal the back, how deep each flap folds, and whether the packet
// fits the postal limits.
func measure(p Params, panels []Panel, sheetPanel Polygon) Metrics {
	front := Rect{X: -p.PanelWidth / 2, Y: -p.PanelHeight / 2, Width: p.PanelWidth, Height: p.PanelHeight}
	heightIn, lengthIn := p.PacketInches()

	m := Metrics{
		PacketWidthMM:  p.PanelWidth,
		PacketHeightMM: p.PanelHeight,
		HeightIn:       heightIn,
		LengthIn:       lengthIn,
		WithinLimits:   p.WithinLimits(),
		FlapDepthMM:    map[string]float64{},
		SheetAreaMM2:   sheetPanel.Area(),
	}

	byID := make(map[string]Panel, len(panels))
	for _, panel := range panels {
		byID[panel.ID] = panel
		m.PartitionAreaMM2 += panel.Polygon.Area()

		if panel.ID == PanelFront {
			continue
		}
		min, max := panel.Polygon.Bounds()
		switch panel.ID {
		case PanelLeft:
			m.FlapDepthMM[panel.ID] = -p.PanelWidth/2 - min.X
		case PanelRight:
			m.FlapDepthMM[panel.ID] = max.X - p.PanelWidth/2
		case PanelTop:
			m.FlapDepthMM[panel.ID] = max.Y - p.PanelHeight/2
		case PanelBottom:
			m.FlapDepthMM[panel.ID] = -p.PanelHeight/2 - min.Y
		}
	}

	// Fold every panel behind the front and keep the footprints.
	footprints := make([]Polygon, 0, len(panels))
	for _, panel := range panels {
		if panel.ID == PanelFront {
			continue
		}
		footprint := foldFootprint(panel, byID)
		footprints = append(footprints, footprint)
		m.ProtrusionMM = math.Max(m.ProtrusionMM, protrusion(footprint, front))
	}

	m.BackCoverage = coverage(front, footprints)

	// The side flaps overlap the corner tucks by how far they reach past
	// the panel edge. The smallest depth is the weakest seam.
	m.SeamOverlapMM = math.Inf(1)
	for _, id := range []string{PanelLeft, PanelRight, PanelTop, PanelBottom} {
		if depth, ok := m.FlapDepthMM[id]; ok {
			m.SeamOverlapMM = math.Min(m.SeamOverlapMM, depth)
		}
	}
	if math.IsInf(m.SeamOverlapMM, 1) {
		m.SeamOverlapMM = 0
	}

	m.Warnings = append(m.Warnings, foldWarnings(p, m)...)
	return m
}

// foldFootprint returns where a panel ends up at t = 1.
//
// A panel is reflected across its own hinge and then across every
// ancestor's hinge, in order. Two reflections compose to the rotation
// the hinge tree performs, and because every fold is a half-turn the
// result stays in the plane - which is exactly why the folded packet is
// flat.
func foldFootprint(panel Panel, byID map[string]Panel) Polygon {
	poly := panel.Polygon
	cur := panel
	for {
		if cur.Hinge == nil {
			break
		}
		poly = reflectPolygon(poly, cur.Hinge.A, cur.Hinge.B)
		if cur.Parent == "" {
			break
		}
		next, ok := byID[cur.Parent]
		if !ok {
			break
		}
		cur = next
	}
	return poly
}

// reflectPolygon mirrors the polygon across the line a-b in the z = 0
// plane.
func reflectPolygon(poly Polygon, a, b Vec3) Polygon {
	out := make(Polygon, len(poly))
	for i, v := range poly {
		out[i] = reflectPoint(v, Vec2{X: a.X, Y: a.Y}, Vec2{X: b.X, Y: b.Y})
	}
	return out
}

// reflectPoint mirrors pt across the line a-b.
func reflectPoint(pt, a, b Vec2) Vec2 {
	d := b.Sub(a)
	len2 := d.Dot(d)
	if len2 < Epsilon {
		return pt
	}
	t := pt.Sub(a).Dot(d) / len2
	proj := a.Add(d.Mul(t))
	return proj.Mul(2).Sub(pt)
}

// coverage samples the front panel's back and reports the fraction of it
// that at least one folded flap covers.
//
// A one millimetre grid is plenty: the metric exists to tell the user
// whether the packet is sealed, not to measure a precise area, and the
// sampling is fast enough to run on every parameter change.
func coverage(front Rect, footprints []Polygon) float64 {
	const step = 1.0
	var total, covered int

	type box struct{ min, max Vec2 }
	boxes := make([]box, len(footprints))
	for i, fp := range footprints {
		min, max := fp.Bounds()
		boxes[i] = box{min: min, max: max}
	}

	for y := front.Y + step/2; y < front.Y+front.Height; y += step {
		for x := front.X + step/2; x < front.X+front.Width; x += step {
			total++
			pt := Vec2{X: x, Y: y}
			for i, fp := range footprints {
				b := boxes[i]
				if pt.X < b.min.X || pt.X > b.max.X || pt.Y < b.min.Y || pt.Y > b.max.Y {
					continue
				}
				if fp.Contains(pt) {
					covered++
					break
				}
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total)
}

// protrusion returns how far the polygon reaches beyond the rectangle.
func protrusion(poly Polygon, r Rect) float64 {
	var worst float64
	for _, v := range poly {
		worst = math.Max(worst, math.Max(r.X-v.X, v.X-(r.X+r.Width)))
		worst = math.Max(worst, math.Max(r.Y-v.Y, v.Y-(r.Y+r.Height)))
	}
	return math.Max(worst, 0)
}

// foldWarnings turns the metrics into the messages the UI shows.
func foldWarnings(p Params, m Metrics) []string {
	var warnings []string

	if !m.WithinLimits {
		warnings = append(warnings, fmt.Sprintf(
			"Packet %.2f x %.2f in is outside the configured letter range (height %.2f-%.2f in, length %.2f-%.2f in).",
			m.HeightIn, m.LengthIn,
			p.Limits.MinHeightMM/MillimetresPerInch, p.Limits.MaxHeightMM/MillimetresPerInch,
			p.Limits.MinLengthMM/MillimetresPerInch, p.Limits.MaxLengthMM/MillimetresPerInch))
	}

	if m.BackCoverage < 0.995 {
		warnings = append(warnings, fmt.Sprintf(
			"The flaps cover %.1f%% of the back; the letter would show through at the edges.",
			m.BackCoverage*100))
	}

	if m.ProtrusionMM > p.TuckDepth+0.6 {
		warnings = append(warnings, fmt.Sprintf(
			"A flap reaches %.1f mm past the packet edge. Move the panel or reduce the rotation.",
			m.ProtrusionMM))
	}

	if m.SeamOverlapMM+0.01 < p.SeamOverlap {
		warnings = append(warnings, fmt.Sprintf(
			"The side flaps overlap by only %.1f mm, below the required %.1f mm.",
			m.SeamOverlapMM, p.SeamOverlap))
	}

	if math.Abs(m.PartitionAreaMM2-m.SheetAreaMM2) > 0.5 {
		warnings = append(warnings, fmt.Sprintf(
			"The panel areas (%.1f mm2) do not add up to the sheet (%.1f mm2).",
			m.PartitionAreaMM2, m.SheetAreaMM2))
	}

	return warnings
}
