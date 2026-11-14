package export

import (
	"bytes"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/incurl/parchment/internal/fold"
	"github.com/incurl/parchment/internal/letters"
)

// TestLetterPDFIsTwoA4Pages checks the exported document's shape.
func TestLetterPDFIsTwoA4Pages(t *testing.T) {
	scene, err := fold.Build(fold.DefaultParams())
	if err != nil {
		t.Fatalf("build fold: %v", err)
	}
	data := LetterPDF(scene, letters.DemoLetter(), Options{})

	if !bytes.HasPrefix(data, []byte("%PDF-1.4")) {
		t.Fatal("missing the PDF header")
	}
	if !bytes.Contains(data, []byte("/Count 2")) {
		t.Fatal("the document does not have two pages")
	}
	if !bytes.Contains(data, []byte("/MediaBox [0 0 595.276 841.890]")) {
		t.Fatal("the pages are not A4")
	}
}

// TestLetterPDFPrintsTheDiagonal proves the diagonal is in the output:
// the address is printed at the computed rotation and the letter at its
// mirror on the other page.
func TestLetterPDFPrintsTheDiagonal(t *testing.T) {
	scene, err := fold.Build(fold.DefaultParams())
	if err != nil {
		t.Fatalf("build fold: %v", err)
	}
	data := string(LetterPDF(scene, letters.DemoLetter(), Options{}))

	angles := textAngles([]byte(data))
	rotation := scene.DerivedRotationDeg()
	if !containsAngle(angles, -rotation) {
		t.Errorf("no text is printed at %.2f degrees; found %v", -rotation, angles)
	}
	if !containsAngle(angles, rotation) {
		t.Errorf("no text is printed at %.2f degrees; found %v", rotation, angles)
	}
	if !strings.Contains(data, "Avery") {
		t.Error("the recipient name is missing from the export")
	}
}

// TestLetterPDFFoldOverlay checks the optional overlay adds content.
func TestLetterPDFFoldOverlay(t *testing.T) {
	scene, err := fold.Build(fold.DefaultParams())
	if err != nil {
		t.Fatalf("build fold: %v", err)
	}
	plain := LetterPDF(scene, letters.DemoLetter(), Options{})
	overlay := LetterPDF(scene, letters.DemoLetter(), Options{ShowFoldLines: true})
	if len(overlay) <= len(plain) {
		t.Errorf("the overlay did not add any drawing: %d vs %d bytes", len(overlay), len(plain))
	}
}

// textAngles extracts every text matrix's angle from the page streams.
func textAngles(data []byte) []float64 {
	re := regexp.MustCompile(`([-0-9.]+) ([-0-9.]+) ([-0-9.]+) ([-0-9.]+) ([-0-9.]+) ([-0-9.]+) Tm`)
	var out []float64
	for _, match := range re.FindAllStringSubmatch(string(data), -1) {
		a, errA := strconv.ParseFloat(match[1], 64)
		b, errB := strconv.ParseFloat(match[2], 64)
		if errA != nil || errB != nil {
			continue
		}
		out = append(out, math.Atan2(b, a)*180/math.Pi)
	}
	return out
}

// containsAngle reports whether an angle is present within a tolerance.
func containsAngle(angles []float64, want float64) bool {
	for _, angle := range angles {
		if math.Abs(angle-want) < 0.05 {
			return true
		}
	}
	return false
}
