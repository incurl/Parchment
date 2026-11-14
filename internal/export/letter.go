// Package export renders a letter into a print-ready PDF.
//
// The output is two A4 pages: Side A (the letter, which ends up inside)
// and Side B (the envelope face, which ends up outside). Both pages use
// the layout geometry the fold engine produced, so the address is printed
// at the computed diagonal and lands upright on the folded packet.
package export

import (
	"github.com/incurl/parchment/internal/fold"
	"github.com/incurl/parchment/internal/letters"
	"github.com/incurl/parchment/internal/pdf"
)

// Options control what the exported pages contain.
type Options struct {
	// ShowFoldLines adds the fold-plan overlay to Side B as light grey
	// dashed lines. The default export is clean; the creases are for a
	// fold-it-yourself sheet.
	ShowFoldLines bool
}

// LetterPDF renders the letter into a PDF.
func LetterPDF(scene fold.Scene, letter letters.Letter, opts Options) []byte {
	doc := pdf.New()

	drawSideA(doc.AddPage(), scene, letter)
	drawSideB(doc.AddPage(), scene, letter, opts)

	return doc.Bytes()
}

// drawSideA renders the letter on the inside of the sheet.
//
// The printed page is the mirror of the sheet frame: the two sides of one
// sheet are mirror images. Mirroring the computed placement back turns
// the letter's baseline into the printed one at minus the panel rotation,
// which is why the printed letter is nearly upright even though the
// address is printed at plus the rotation on the other side.
func drawSideA(page *pdf.Page, scene fold.Scene, letter letters.Letter) {
	placement := scene.LetterPlacement
	rect := scene.Layout.LetterRect
	size := scene.Layout.BodyFontMM
	lineHeight := scene.Layout.LineHeightMM

	// The page is the sheet seen from Side A.
	toPage := func(p fold.Vec2) fold.Vec2 {
		return fold.Vec2{X: scene.Sheet.Width - p.X, Y: p.Y}
	}
	direction := fold.Vec2{X: -placement.XAxis.X, Y: placement.XAxis.Y}
	rotation := angleDeg(direction)

	lines := wrap(letter.SideA, rect.Width, size, false)
	y := rect.Y + size
	for _, line := range lines {
		at := toPage(placement.Apply(fold.Vec2{X: rect.X, Y: y}))
		page.Text(at.X, at.Y, size, line, rotation, false)
		y += lineHeight
	}
}
