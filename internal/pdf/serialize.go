package pdf

import (
	"bytes"
	"fmt"
	"strings"
)

// PageCount returns the number of pages added so far.
func (d *Doc) PageCount() int { return len(d.pages) }

// Bytes serialises the document.
//
// The file is built directly rather than through a library so the page
// size, the fonts and the byte offsets are all visible in one place. The
// cross-reference table is written from the real offsets, so a PDF reader
// opens the file without repair.
func (d *Doc) Bytes() []byte {
	objects := make([]string, 0, 4+2*len(d.pages))

	// 1: catalog, 2: page tree, 3-4: the two standard fonts.
	objects = append(objects, "<< /Type /Catalog /Pages 2 0 R >>")

	kids := make([]string, len(d.pages))
	for i := range d.pages {
		// Page objects are 5, 7, 9... and each content stream
		// follows its page.
		kids[i] = fmt.Sprintf("%d 0 R", 5+2*i)
	}
	objects = append(objects, fmt.Sprintf(
		"<< /Type /Pages /Kids [%s] /Count %d >>",
		strings.Join(kids, " "), len(d.pages)))
	objects = append(objects,
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>",
	)

	width := A4WidthMM * PointsPerMM
	height := A4HeightMM * PointsPerMM

	for i, page := range d.pages {
		pageObject := 5 + 2*i
		contentObject := pageObject + 1
		objects = append(objects, fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.3f %.3f] "+
				"/Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>",
			width, height, contentObject))

		content := page.ops.String()
		objects = append(objects, fmt.Sprintf(
			"<< /Length %d >>\nstream\n%sendstream", len(content), content))
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	// A binary comment marks the file as binary for transfer tools.
	buf.WriteString("%\xE2\xE3\xCF\xD3\n")

	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}

	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, xref)

	return buf.Bytes()
}
