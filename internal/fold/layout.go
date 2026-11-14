package fold

// Layout is the geometry of the printed content, in panel-frame
// millimetres.
//
// It is part of the scene so that the 2D preview, the 3D textures and
// the PDF exporter place the address, the return address and the stamp
// identically. Nothing here is a hard-coded screen coordinate: the
// rectangles are derived from the panel size, and every renderer maps
// them onto the sheet with FrontPanel.TransformToFlat.
type Layout struct {
	// AddressRect is the recipient block. The text is right-aligned
	// inside it, as the postal service reads it.
	AddressRect Rect `json:"address_rect"`
	// ReturnRect is the return address in the top-left corner.
	ReturnRect Rect `json:"return_rect"`
	// StampRect is where the stamp goes, top-right.
	StampRect Rect `json:"stamp_rect"`
	// LetterRect is the safe area for the letter body on Side A. Text
	// that leaves it is in the fold margin and is warned about.
	LetterRect Rect `json:"letter_rect"`

	// Font sizes and the line advance, all in millimetres.
	AddressFontMM float64 `json:"address_font_mm"`
	ReturnFontMM  float64 `json:"return_font_mm"`
	BodyFontMM    float64 `json:"body_font_mm"`
	LineHeightMM  float64 `json:"line_height_mm"`

	// RotationDeg is the angle the content is printed at on Side B. It
	// is the front panel's rotation, quoted here so a renderer can
	// report it next to the layout.
	RotationDeg float64 `json:"rotation_deg"`
}

// makeLayout derives the print layout from the panel size.
func makeLayout(p Params) Layout {
	const (
		margin     = 12.0
		stampW     = 22.0
		stampH     = 26.0
		addressFnt = 3.6
		returnFnt  = 2.6
		bodyFnt    = 3.2
		lineHeight = 4.6
		safeInset  = 8.0
	)

	return Layout{
		AddressRect: Rect{
			X:      p.PanelWidth * 0.38,
			Y:      p.PanelHeight * 0.50,
			Width:  p.PanelWidth * 0.58,
			Height: p.PanelHeight * 0.40,
		},
		ReturnRect: Rect{
			X:      margin,
			Y:      margin,
			Width:  p.PanelWidth * 0.46,
			Height: lineHeight * 3,
		},
		StampRect: stampRect(p, margin, stampW, stampH),
		LetterRect: Rect{
			X:      safeInset,
			Y:      safeInset,
			Width:  p.PanelWidth - 2*safeInset,
			Height: p.PanelHeight - 2*safeInset,
		},
		AddressFontMM: addressFnt,
		ReturnFontMM:  returnFnt,
		BodyFontMM:    bodyFnt,
		LineHeightMM:  lineHeight,
		RotationDeg:   p.RotationDeg,
	}
}

// stampRect returns the stamp rectangle for the chosen corner.
func stampRect(p Params, margin, w, h float64) Rect {
	y := margin
	if p.StampCorner == "bottom-right" {
		y = p.PanelHeight - margin - h
	}
	if p.StampCorner == "top-left" {
		return Rect{X: margin, Y: y, Width: w, Height: h}
	}
	return Rect{X: p.PanelWidth - margin - w, Y: y, Width: w, Height: h}
}

// creases returns the fold lines the 2D overlay draws, in sheet
// coordinates.
//
// The front panel's outline is the most important line: it is the
// boundary of the clean face and the region the address must stay
// inside. The corner creases are the diagonal folds, which are parallel
// to the panel's edges and therefore visibly diagonal on the sheet.
func creases(p Params, frame Frame, panels []Panel) []Crease {
	var out []Crease

	for _, panel := range panels {
		if panel.ID != PanelFront {
			continue
		}
		for i := range panel.SheetPolygon {
			out = append(out, Crease{
				ID:   "edge-" + itoa(i),
				A:    panel.SheetPolygon[i],
				B:    panel.SheetPolygon[(i+1)%len(panel.SheetPolygon)],
				Kind: "edge",
			})
		}
	}

	for _, panel := range panels {
		if panel.Hinge == nil || panel.Parent != PanelLeft && panel.Parent != PanelRight {
			continue
		}
		out = append(out, Crease{
			ID:   "crease-" + panel.ID,
			A:    frame.PanelToSheet(fromWorld(panel.Hinge.A, p)),
			B:    frame.PanelToSheet(fromWorld(panel.Hinge.B, p)),
			Kind: "tuck",
		})
	}

	// The letter's safe area, so the overlay shows where the text may
	// run without entering the fold margin.
	rect := makeLayout(p).LetterRect.Polygon()
	for i := range rect {
		out = append(out, Crease{
			ID:   "safe-" + itoa(i),
			A:    frame.PanelToSheet(rect[i]),
			B:    frame.PanelToSheet(rect[(i+1)%len(rect)]),
			Kind: "safe",
		})
	}

	return out
}

// fromWorld maps a world point back into the panel frame.
func fromWorld(v Vec3, p Params) Vec2 {
	return Vec2{X: v.X + p.PanelWidth/2, Y: p.PanelHeight/2 - v.Y}
}

// itoa formats a small non-negative integer without pulling strconv in
// just for a crease identifier.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [4]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
