package export

import (
	"strings"

	"github.com/incurl/parchment/internal/fold"
	"github.com/incurl/parchment/internal/letters"
	"github.com/incurl/parchment/internal/pdf"
)

// drawSideB renders the envelope face on the outside of the sheet.
func drawSideB(page *pdf.Page, scene fold.Scene, letter letters.Letter, opts Options) {
	placement := scene.SheetPlacement
	layout := scene.Layout

	// The recipient block is right-aligned: the right edge is the anchor
	// a postal reader's eye follows.
	address := []struct {
		text string
		bold bool
	}{
		{letter.RecipientName, true},
		{letter.RecipientStreet, false},
		{joinCityLine(letter.RecipientCity, letter.RecipientState, letter.RecipientZIP), false},
	}
	y := layout.AddressRect.Y + layout.AddressFontMM
	for _, line := range address {
		if strings.TrimSpace(line.text) == "" {
			continue
		}
		size := layout.AddressFontMM
		if !line.bold {
			size = layout.AddressFontMM - 0.4
		}
		width := pdf.TextWidth(line.text, size, line.bold)
		x := layout.AddressRect.X + layout.AddressRect.Width - width
		at := placement.Apply(fold.Vec2{X: x, Y: y})
		page.Text(at.X, at.Y, size, line.text, placement.RotationDeg, line.bold)
		y += layout.LineHeightMM
	}

	// The return address.
	returnLines := []string{
		letter.ReturnName,
		letter.ReturnStreet,
		joinCityLine(letter.ReturnCity, letter.ReturnState, letter.ReturnZIP),
	}
	y = layout.ReturnRect.Y + layout.ReturnFontMM
	for _, line := range returnLines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		at := placement.Apply(fold.Vec2{X: layout.ReturnRect.X, Y: y})
		page.Text(at.X, at.Y, layout.ReturnFontMM, line, placement.RotationDeg, false)
		y += layout.ReturnFontMM * 1.5
	}

	drawStamp(page, scene)

	if opts.ShowFoldLines {
		page.SetStrokeGray(0.55)
		for _, crease := range scene.Creases {
			page.Line(crease.A.X, crease.A.Y, crease.B.X, crease.B.Y, 0.12)
		}
		page.SetStrokeGray(0)
	}
}

// drawStamp outlines the stamp area.
//
// The outline is a guide, not part of the design: it shows where a real
// stamp goes. A printed stamp artwork would replace it.
func drawStamp(page *pdf.Page, scene fold.Scene) {
	placement := scene.SheetPlacement
	printed := placement.ApplyPolygon(scene.Layout.StampRect.Polygon())

	page.SetStrokeGray(0.6)
	for i := range printed {
		a := printed[i]
		b := printed[(i+1)%len(printed)]
		page.Line(a.X, a.Y, b.X, b.Y, 0.12)
	}

	label := "STAMP"
	width := pdf.TextWidth(label, 2.2, false)
	centre := placement.Apply(fold.Vec2{
		X: scene.Layout.StampRect.X + scene.Layout.StampRect.Width/2 - width/2,
		Y: scene.Layout.StampRect.Y + scene.Layout.StampRect.Height/2 + 1,
	})
	page.SetFillGray(0.6)
	page.Text(centre.X, centre.Y, 2.2, label, placement.RotationDeg, false)
	page.SetFillGray(0)
	page.SetStrokeGray(0)
}
