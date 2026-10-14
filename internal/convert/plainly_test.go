package convert_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
)

// minimalPDF is a one-page PDF with no text on it, as a scan is, its
// cross-reference table counted so the reader takes it.
var minimalPDF = func() string {
	objs := []string{"<</Type/Catalog/Pages 2 0 R>>", "<</Type/Pages/Kids[3 0 R]/Count 1>>", "<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]>>"}
	b := "%PDF-1.4\n"
	xref := "xref\n0 4\n0000000000 65535 f \n"
	for i, o := range objs {
		xref += fmt.Sprintf("%010d 00000 n \n", len(b))
		b += fmt.Sprintf("%d 0 obj%sendobj\n", i+1, o)
	}
	return b + xref + "trailer<</Size 4/Root 1 0 R>>\nstartxref\n" + fmt.Sprint(len(b)) + "\n%%EOF\n"
}()

// A file that cannot be read says what happened to it and what the person
// can do, never the parser's own words: locked, damaged, an older Office
// file, a scan.
func TestAFileThatCannotBeReadSaysWhatToDo(t *testing.T) {
	t.Parallel()
	ole := append([]byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}, make([]byte, 504)...)
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"payslip.docx", ole, "locked with a password"},
		{"budget.xlsx", ole, "locked with a password"},
		{"broken.docx", []byte("not a zip at all"), "damaged, or is not really a .docx file"},
		{"cut.pdf", []byte(minimalPDF[:60]), "damaged, or is not really a .pdf file"},
		{"old.doc", ole, "Open it in Word, save it as .docx"},
		{"old.xls", ole, "Open it in Excel, save it as .xlsx"},
		{"scan.pdf", []byte(minimalPDF), "its pages are pictures"},
	}
	for _, c := range cases {
		_, err := convert.Read(c.name, c.data)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: want %q, got %v", c.name, c.want, err)
			continue
		}
		for _, jargon := range []string{"zip:", "%EOF", "OCR", "converter", "xml"} {
			if strings.Contains(err.Error(), jargon) {
				t.Errorf("%s: says %q: %v", c.name, jargon, err)
			}
		}
	}
}

// A PDF saved with another ending is read as the PDF it is.
func TestAPDFUnderAnotherNameIsRead(t *testing.T) {
	t.Parallel()
	_, pdfErr := convert.Read("scan.pdf", []byte(minimalPDF))
	res, err := convert.Read("letter.docx", []byte(minimalPDF))
	if res.Kind != "pdf" || err == nil || err.Error() != pdfErr.Error() {
		t.Errorf("a PDF named .docx is read as a PDF: kind %q, %v", res.Kind, err)
	}
}

// Text saved the Windows way still reads: Notepad's UTF-16, Excel's
// Windows-1252 CSV, and a UTF-8 mark that is not part of the first word.
func TestTextReadsHoweverItWasSaved(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"prices.csv", []byte("Item,Cost\nCaf\xe9,3 \xa3\n"), "| Café | 3 £ |"},
		{"note.txt", []byte{0xff, 0xfe, 'H', 0, 0xe9, 0, 'l', 0, 'o', 0}, "Hélo"},
		{"mark.txt", []byte("\xef\xbb\xbfFirst"), "First"},
	}
	for _, c := range cases {
		res, err := convert.Read(c.name, c.data)
		if err != nil || !strings.HasPrefix(res.Markdown, c.want) && !strings.Contains(res.Markdown, c.want) {
			t.Errorf("%s: want %q, got %q (%v)", c.name, c.want, res.Markdown, err)
		}
		if strings.ContainsRune(res.Markdown, 0xfffd) || strings.HasPrefix(res.Markdown, string(rune(0xfeff))) {
			t.Errorf("%s: broken characters: %q", c.name, res.Markdown)
		}
	}
}
