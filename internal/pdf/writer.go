// Package pdf writes the printable output: a minimal, dependency-free
// PDF 1.4 writer.
//
// The application exports two A4 pages at true scale. A PDF writer that
// understands only lines, rectangles and the two standard Helvetica
// fonts is a few hundred lines, keeps the build free of a new
// dependency, and guarantees the page size: the MediaBox is written as
// 595.276 x 841.89 points, which is 210 x 297 mm exactly (72 points per
// inch, 25.4 mm per inch).
//
// Coordinates are given in millimetres from the page's top-left corner,
// like every other frame in this application, and converted here. The
// conversion is the only place the PDF's bottom-left origin appears.
package pdf

import (
	"fmt"
	"math"
	"strings"
)

// Page size in millimetres.
const (
	A4WidthMM  = 210.0
	A4HeightMM = 297.0
)

// PointsPerMM is the exact conversion at 72 points per inch.
const PointsPerMM = 72.0 / 25.4

// Doc is a PDF document under construction.
type Doc struct {
	pages []*Page
}

// Page is a single page's content stream.
type Page struct {
	ops strings.Builder
}

// New creates an empty A4 document.
func New() *Doc { return &Doc{} }

// AddPage starts a new A4 page.
func (d *Doc) AddPage() *Page {
	p := &Page{}
	d.pages = append(d.pages, p)
	return p
}

// Text draws a single line of text with its baseline at (x, y) mm from
// the page's top-left corner, rotated clockwise by rotationDeg.
//
// The size is in millimetres so the caller can keep one set of layout
// numbers for the canvas preview and the PDF.
func (p *Page) Text(x, y, sizeMM float64, text string, rotationDeg float64, bold bool) {
	if text == "" {
		return
	}

	rad := rotationDeg * math.Pi / 180
	c, s := math.Cos(rad), math.Sin(rad)

	// Sheet (y down) to PDF (y up): a direction (dx, dy) becomes
	// (dx, -dy), so the text matrix's counter-clockwise angle is the
	// negative of the clockwise one.
	px := x * PointsPerMM
	py := (A4HeightMM - y) * PointsPerMM

	font := "F1"
	if bold {
		font = "F2"
	}

	fmt.Fprintf(&p.ops, "BT /%s %0.3f Tf %0.5f %0.5f %0.5f %0.5f %0.3f %0.3f Tm (%s) Tj ET\n",
		font, sizeMM*PointsPerMM, c, -s, s, c, px, py, escape(text))
}

// TextWidth returns the width of text at the given size, in millimetres.
func TextWidth(text string, sizeMM float64, bold bool) float64 {
	table := helveticaWidths
	if bold {
		table = helveticaBoldWidths
	}
	var sum int
	for _, r := range text {
		if r >= 32 && r <= 126 {
			sum += table[r-32]
		} else {
			sum += table['?'-32]
		}
	}
	return float64(sum) / 1000 * sizeMM
}

// Line draws a straight line.
func (p *Page) Line(x1, y1, x2, y2, widthMM float64) {
	if widthMM <= 0 {
		widthMM = 0.1
	}
	fmt.Fprintf(&p.ops, "%0.3f w %0.3f %0.3f m %0.3f %0.3f l S\n",
		widthMM*PointsPerMM,
		x1*PointsPerMM, (A4HeightMM-y1)*PointsPerMM,
		x2*PointsPerMM, (A4HeightMM-y2)*PointsPerMM)
}

// Rect draws a rectangle outline.
func (p *Page) Rect(x, y, width, height, lineWidthMM float64) {
	if lineWidthMM <= 0 {
		lineWidthMM = 0.1
	}
	fmt.Fprintf(&p.ops, "%0.3f w %0.3f %0.3f %0.3f %0.3f re S\n",
		lineWidthMM*PointsPerMM,
		x*PointsPerMM, (A4HeightMM-y-height)*PointsPerMM,
		width*PointsPerMM, height*PointsPerMM)
}

// DashedRect draws a dashed rectangle outline, used for the fold-plan
// overlay.
func (p *Page) DashedRect(x, y, width, height, lineWidthMM float64) {
	if lineWidthMM <= 0 {
		lineWidthMM = 0.1
	}
	fmt.Fprintf(&p.ops, "[%0.2f %0.2f] 0 d %0.3f w %0.3f %0.3f %0.3f %0.3f re S [] 0 d\n",
		2*PointsPerMM, 1.5*PointsPerMM,
		lineWidthMM*PointsPerMM,
		x*PointsPerMM, (A4HeightMM-y-height)*PointsPerMM,
		width*PointsPerMM, height*PointsPerMM)
}

// escape protects the characters that delimit a PDF string.
func escape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '(', ')', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\r':
			// ignore
		default:
			b.WriteRune(encodeWinAnsi(r))
		}
	}
	return b.String()
}

// encodeWinAnsi maps a rune onto the WinAnsiEncoding byte the standard
// fonts expect. Text outside the mapping becomes '?', because a silent
// substitution is better than a corrupt string.
func encodeWinAnsi(r rune) rune {
	if r < 128 {
		return r
	}
	if b, ok := winAnsiHigh[r]; ok {
		return b
	}
	return '?'
}

// winAnsiHigh maps the characters a letter is likely to use onto
// WinAnsiEncoding.
var winAnsiHigh = map[rune]rune{
	'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85,
	'†': 0x86, '‡': 0x87, 'ˆ': 0x88, '‰': 0x89, 'Š': 0x8A,
	'‹': 0x8B, 'Œ': 0x8C, '‘': 0x91, '’': 0x92, '“': 0x93,
	'”': 0x94, '•': 0x95, '–': 0x96, '—': 0x97, '™': 0x99,
	'›': 0x9B, 'œ': 0x9C, '¡': 0xA1, '¢': 0xA2, '£': 0xA3,
	'¤': 0xA4, '¥': 0xA5, '¦': 0xA6, '§': 0xA7, '¨': 0xA8,
	'©': 0xA9, 'ª': 0xAA, '«': 0xAB, '¬': 0xAC, '®': 0xAE,
	'¯': 0xAF, '°': 0xB0, '±': 0xB1, '²': 0xB2, '³': 0xB3,
	'´': 0xB4, 'µ': 0xB5, '¶': 0xB6, '·': 0xB7, '¸': 0xB8,
	'¹': 0xB9, 'º': 0xBA, '»': 0xBB, '¼': 0xBC, '½': 0xBD,
	'¾': 0xBE, '¿': 0xBF, 'À': 0xC0, 'Á': 0xC1, 'Â': 0xC2,
	'Ã': 0xC3, 'Ä': 0xC4, 'Å': 0xC5, 'Æ': 0xC6, 'Ç': 0xC7,
	'È': 0xC8, 'É': 0xC9, 'Ê': 0xCA, 'Ë': 0xCB, 'Ì': 0xCC,
	'Í': 0xCD, 'Î': 0xCE, 'Ï': 0xCF, 'Ð': 0xD0, 'Ñ': 0xD1,
	'Ò': 0xD2, 'Ó': 0xD3, 'Ô': 0xD4, 'Õ': 0xD5, 'Ö': 0xD6,
	'×': 0xD7, 'Ø': 0xD8, 'Ù': 0xD9, 'Ú': 0xDA, 'Û': 0xDB,
	'Ü': 0xDC, 'Ý': 0xDD, 'Þ': 0xDE, 'ß': 0xDF, 'à': 0xE0,
	'á': 0xE1, 'â': 0xE2, 'ã': 0xE3, 'ä': 0xE4, 'å': 0xE5,
	'æ': 0xE6, 'ç': 0xE7, 'è': 0xE8, 'é': 0xE9, 'ê': 0xEA,
	'ë': 0xEB, 'ì': 0xEC, 'í': 0xED, 'î': 0xEE, 'ï': 0xEF,
	'ð': 0xF0, 'ñ': 0xF1, 'ò': 0xF2, 'ó': 0xF3, 'ô': 0xF4,
	'õ': 0xF5, 'ö': 0xF6, '÷': 0xF7, 'ø': 0xF8, 'ù': 0xF9,
	'ú': 0xFA, 'û': 0xFB, 'ü': 0xFC, 'ý': 0xFD, 'þ': 0xFE,
	'ÿ': 0xFF,
}
