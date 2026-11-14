package pdf

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
)

// TestDocumentIsWellFormed checks the file a reader has to accept: the
// header, the page tree, the A4 media box, a working xref and the
// trailer.
func TestDocumentIsWellFormed(t *testing.T) {
	doc := New()
	page := doc.AddPage()
	page.Text(20, 20, 4, "Hello (paper)", 0, false)
	doc.AddPage().Line(10, 10, 200, 287, 0.2)

	if doc.PageCount() != 2 {
		t.Fatalf("PageCount = %d, want 2", doc.PageCount())
	}

	data := doc.Bytes()
	if !bytes.HasPrefix(data, []byte("%PDF-1.4")) {
		t.Fatal("missing the PDF header")
	}
	if !bytes.HasSuffix(bytes.TrimSpace(data), []byte("%%EOF")) {
		t.Fatal("missing the end-of-file marker")
	}
	if !bytes.Contains(data, []byte("/Count 2")) {
		t.Fatal("the page tree does not count two pages")
	}
	if !bytes.Contains(data, []byte("/MediaBox [0 0 595.276 841.890]")) {
		t.Fatalf("the media box is not A4: %s", mediaBox(data))
	}
	if !bytes.Contains(data, []byte(`Hello \(paper\)`)) {
		t.Fatal("the parentheses in the text were not escaped")
	}

	// The xref offset must point at the xref table.
	xref := strings.Index(string(data), "startxref")
	if xref < 0 {
		t.Fatal("no startxref")
	}
	var offset int
	if _, err := fmt.Sscanf(string(data[xref:]), "startxref\n%d", &offset); err != nil {
		t.Fatalf("parse startxref: %v", err)
	}
	if !bytes.HasPrefix(data[offset:], []byte("xref")) {
		t.Fatalf("startxref points at %q", data[offset:offset+8])
	}
}

// mediaBox returns the MediaBox line for the failure message.
func mediaBox(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "MediaBox") {
			return line
		}
	}
	return "none"
}

// TestTextWidthMatchesTheMetrics checks the width table against a known
// Helvetica advance: "Hello" at 10 pt is 22.78 pt wide.
func TestTextWidthMatchesTheMetrics(t *testing.T) {
	// H=722, e=556, l=222, l=222, o=556 -> 2278/1000 em.
	got := TextWidth("Hello", 10, false)
	if math.Abs(got-22.78) > 0.01 {
		t.Errorf("TextWidth(\"Hello\") = %.2f, want 22.78", got)
	}
	if bold := TextWidth("Hello", 10, true); bold <= got {
		t.Errorf("the bold width %.2f is not wider than the regular %.2f", bold, got)
	}
}

// TestRotationIsWrittenToTheTextMatrix checks the coordinate conversion:
// a clockwise rotation in the sheet frame becomes a negative angle in the
// PDF's counter-clockwise frame.
func TestRotationIsWrittenToTheTextMatrix(t *testing.T) {
	doc := New()
	page := doc.AddPage()
	page.Text(10, 10, 4, "tilted", 15, false)

	data := string(doc.Bytes())
	idx := strings.Index(data, "Tm")
	if idx < 0 {
		t.Fatal("no text matrix")
	}
	start := strings.LastIndex(data[:idx], "Tf")
	fields := strings.Fields(data[start:idx])
	if len(fields) < 6 {
		t.Fatalf("malformed text matrix: %q", data[start:idx])
	}
	// fields: Tf a b c d e f
	a := parseFloat(t, fields[1])
	b := parseFloat(t, fields[2])
	angle := math.Atan2(b, a) * 180 / math.Pi
	if math.Abs(angle+15) > 0.01 {
		t.Errorf("text matrix angle = %.3f, want -15", angle)
	}
}

// parseFloat parses a PDF operator operand.
func parseFloat(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return v
}
