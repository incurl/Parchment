package fold

import (
	"fmt"
	"math"
)

// Params are the geometric inputs of the fold engine.
//
// Everything the application shows is derived from these values: the
// panel polygons, the hinges, the fold schedule, the address rotation
// and the packet dimensions. Changing a parameter changes all of them
// together, which is what keeps the address upright when the panel size
// or the fold angle changes.
type Params struct {
	// SheetWidth and SheetHeight are the flat sheet in millimetres.
	// A4 is 210 x 297, and the exporter relies on it.
	SheetWidth  float64 `json:"sheet_width"`
	SheetHeight float64 `json:"sheet_height"`

	// PanelWidth and PanelHeight are the visible front panel in
	// millimetres. The folded packet is this rectangle.
	PanelWidth  float64 `json:"panel_width"`
	PanelHeight float64 `json:"panel_height"`

	// RotationDeg is the angle between the front panel and the sheet.
	// It is the diagonal of the fold: the address is printed at this
	// angle on Side B and reads upright on the packet.
	RotationDeg float64 `json:"rotation_deg"`

	// OffsetX and OffsetY move the panel's centre away from the sheet's
	// centre, in millimetres. They let the user bias the flaps: a panel
	// pushed towards the top leaves a deeper final flap at the bottom.
	OffsetX float64 `json:"offset_x"`
	OffsetY float64 `json:"offset_y"`

	// TuckDepth is how far past the front panel's edge the corner tucks
	// are cut. The crease of a corner tuck is parallel to the panel's
	// edge and TuckDepth away from it, so the folded tuck lands inside
	// the packet. Three millimetres is enough to hide the sheet's
	// corner without touching the letter area.
	TuckDepth float64 `json:"tuck_depth"`

	// SeamOverlap is the minimum overlap the side flaps must have over
	// the corner tucks, in millimetres. It is a validation threshold,
	// not a geometry input.
	SeamOverlap float64 `json:"seam_overlap"`

	// StampCorner is where the stamp is printed on the envelope face:
	// "top-right" (the default, and the corner USPS expects),
	// "top-left" or "bottom-right".
	StampCorner string `json:"stamp_corner"`

	// Limits are the postal limits the packet is validated against.
	Limits Limits `json:"limits"`
}

// Limits are the standard letter dimensions in millimetres.
//
// The default is the USPS letter range: height 3.5 to 6.125 inches and
// length 5 to 11.5 inches. Length is the longer side and height the
// shorter one.
type Limits struct {
	MinHeightMM float64 `json:"min_height_mm"`
	MaxHeightMM float64 `json:"max_height_mm"`
	MinLengthMM float64 `json:"min_length_mm"`
	MaxLengthMM float64 `json:"max_length_mm"`
}

// MillimetresPerInch is the exact conversion.
const MillimetresPerInch = 25.4

// USPSLetterLimits returns the default USPS letter limits, converted to
// millimetres so the rest of the engine stays in one unit.
func USPSLetterLimits() Limits {
	return Limits{
		MinHeightMM: 3.5 * MillimetresPerInch,
		MaxHeightMM: 6.125 * MillimetresPerInch,
		MinLengthMM: 5.0 * MillimetresPerInch,
		MaxLengthMM: 11.5 * MillimetresPerInch,
	}
}

// DefaultParams returns the fold the application ships with.
//
// The panel is 150 x 115 mm, which folds down to a 5.91 x 4.53 in packet
// - inside the USPS letter range - and is rotated by 15 degrees inside
// the A4 sheet. The rotation is what makes the address diagonal, and it
// leaves the final flap exactly deep enough to wrap the packet: with a
// 105 mm panel the same fold overhangs by 13 mm, which the metrics
// report as a warning.
func DefaultParams() Params {
	return Params{
		SheetWidth:  210,
		SheetHeight: 297,
		PanelWidth:  150,
		PanelHeight: 115,
		RotationDeg: 15,
		OffsetX:     0,
		OffsetY:     0,
		TuckDepth:   3,
		SeamOverlap: 5,
		StampCorner: "top-right",
		Limits:      USPSLetterLimits(),
	}
}

// isZero reports whether the limits are unset.
func (l Limits) isZero() bool {
	return l.MinHeightMM == 0 && l.MaxHeightMM == 0 &&
		l.MinLengthMM == 0 && l.MaxLengthMM == 0
}

// WithDefaults fills every unset field with the default value.
//
// The application sends a full Params value, but an older stored draft
// may not have every field, and a zero sheet would produce a degenerate
// scene. Filling the gaps keeps an old draft openable.
func (p Params) WithDefaults() Params {
	d := DefaultParams()
	if p.SheetWidth <= 0 {
		p.SheetWidth = d.SheetWidth
	}
	if p.SheetHeight <= 0 {
		p.SheetHeight = d.SheetHeight
	}
	if p.PanelWidth <= 0 {
		p.PanelWidth = d.PanelWidth
	}
	if p.PanelHeight <= 0 {
		p.PanelHeight = d.PanelHeight
	}
	if p.TuckDepth <= 0 {
		p.TuckDepth = d.TuckDepth
	}
	if p.SeamOverlap <= 0 {
		p.SeamOverlap = d.SeamOverlap
	}
	if p.StampCorner == "" {
		p.StampCorner = d.StampCorner
	}
	if p.Limits.isZero() {
		p.Limits = d.Limits
	}
	return p
}

// Validate rejects parameters the engine cannot fold.
func (p Params) Validate() error {
	switch {
	case p.SheetWidth <= 0 || p.SheetHeight <= 0:
		return fmt.Errorf("fold: the sheet must have a positive size")
	case p.PanelWidth <= 0 || p.PanelHeight <= 0:
		return fmt.Errorf("fold: the front panel must have a positive size")
	case p.PanelWidth >= p.SheetWidth || p.PanelHeight >= p.SheetHeight:
		return fmt.Errorf("fold: the front panel (%0.1f x %0.1f mm) must be smaller than the sheet (%0.1f x %0.1f mm)",
			p.PanelWidth, p.PanelHeight, p.SheetWidth, p.SheetHeight)
	case math.Abs(p.RotationDeg) >= 90:
		return fmt.Errorf("fold: the panel rotation must be smaller than 90 degrees")
	case p.TuckDepth <= 0 || p.TuckDepth >= p.PanelWidth/2 || p.TuckDepth >= p.PanelHeight/2:
		return fmt.Errorf("fold: the tuck depth must be positive and smaller than half the panel")
	case p.SeamOverlap <= 0:
		return fmt.Errorf("fold: the seam overlap must be positive")
	case !validStampCorner(p.StampCorner):
		return fmt.Errorf("fold: the stamp corner must be top-right, top-left or bottom-right")
	case p.Limits.MinHeightMM > p.Limits.MaxHeightMM || p.Limits.MinLengthMM > p.Limits.MaxLengthMM:
		return fmt.Errorf("fold: the postal limits are inverted")
	}
	return nil
}

// validStampCorner reports whether the value is one the layout supports.
func validStampCorner(corner string) bool {
	switch corner {
	case "top-right", "top-left", "bottom-right":
		return true
	}
	return false
}

// Packet returns the folded packet's dimensions in millimetres.
//
// The front panel is one layer of paper, so the packet is exactly the
// panel: the flaps fold behind it and the corner tucks land inside it.
func (p Params) Packet() Size {
	return Size{Width: p.PanelWidth, Height: p.PanelHeight}
}

// PacketInches returns the packet's shorter and longer side in inches.
func (p Params) PacketInches() (height, length float64) {
	short := math.Min(p.PanelWidth, p.PanelHeight)
	long := math.Max(p.PanelWidth, p.PanelHeight)
	return short / MillimetresPerInch, long / MillimetresPerInch
}

// WithinLimits reports whether the packet satisfies the postal limits.
func (p Params) WithinLimits() bool {
	height, length := p.PacketInches()
	return height >= p.Limits.MinHeightMM/MillimetresPerInch &&
		height <= p.Limits.MaxHeightMM/MillimetresPerInch &&
		length >= p.Limits.MinLengthMM/MillimetresPerInch &&
		length <= p.Limits.MaxLengthMM/MillimetresPerInch
}
