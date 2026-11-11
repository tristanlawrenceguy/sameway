package convert_test

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
)

func jpegOf(w, h int) []byte {
	var b bytes.Buffer
	jpeg.Encode(&b, image.NewGray(image.Rect(0, 0, w, h)), nil)
	return b.Bytes()
}

// A scan kept as a PDF gives its pages as the JPEGs they are, in order,
// up to the most asked for; a logo or a stamp is not a page.
func TestAScansPagesAreItsPictures(t *testing.T) {
	t.Parallel()
	one, two, logo := jpegOf(620, 877), jpegOf(877, 620), jpegOf(120, 40)
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n1 0 obj << /Length 9 >>\nstream\nBT ET q Q\nendstream\nendobj\n")
	for _, img := range [][]byte{logo, one, two, one} {
		pdf.WriteString("2 0 obj << /Type /XObject /Subtype /Image /Filter /DCTDecode >>\nstream\r\n")
		pdf.Write(img)
		pdf.WriteString("\r\nendstream\nendobj\n")
	}
	pages := convert.ScannedPages(pdf.Bytes(), 2)
	if len(pages) != 2 || !bytes.Equal(pages[0], one) || !bytes.Equal(pages[1], two) {
		t.Errorf("the two first pages as they are, not the logo: %d", len(pages))
	}
	if got := convert.ScannedPages([]byte("%PDF-1.4 words only"), 3); len(got) != 0 {
		t.Errorf("a PDF of words has no scanned pages: %d", len(got))
	}
}
