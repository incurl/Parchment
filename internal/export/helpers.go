// Text layout helpers shared by the two exported pages.
package export

import (
	"math"
	"strings"

	"github.com/incurl/parchment/internal/fold"
	"github.com/incurl/parchment/internal/pdf"
)

// joinCityLine builds the "City, ST 12345" line.
func joinCityLine(city, state, zip string) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(city))
	if s := strings.TrimSpace(state); s != "" {
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		b.WriteString(s)
	}
	if z := strings.TrimSpace(zip); z != "" {
		if b.Len() > 0 {
			b.WriteString("  ")
		}
		b.WriteString(z)
	}
	return b.String()
}

// angleDeg returns the direction's angle in degrees in the page frame,
// where y points down, so a positive angle is clockwise.
func angleDeg(d fold.Vec2) float64 {
	return d.AngleRad() * 180 / math.Pi
}

// wrap breaks the text into lines that fit the width, honouring the
// author's line breaks.
func wrap(text string, widthMM, sizeMM float64, bold bool) []string {
	var out []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}

		line := words[0]
		for _, word := range words[1:] {
			candidate := line + " " + word
			if pdf.TextWidth(candidate, sizeMM, bold) <= widthMM {
				line = candidate
				continue
			}
			out = append(out, line)
			line = word
		}
		out = append(out, line)
	}
	return out
}
