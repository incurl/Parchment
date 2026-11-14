package pdf

import "fmt"

// SetStrokeGray sets the stroke colour on the grey ramp, 0 black to 1
// white.
func (p *Page) SetStrokeGray(g float64) {
	fmt.Fprintf(&p.ops, "%0.3f G\n", clamp01(g))
}

// SetFillGray sets the fill colour on the grey ramp.
func (p *Page) SetFillGray(g float64) {
	fmt.Fprintf(&p.ops, "%0.3f g\n", clamp01(g))
}

// clamp01 keeps a grey value inside the printable range.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
