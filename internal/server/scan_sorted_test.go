package server_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// scannedPDF is a PDF whose one page is a photo, as a scanner makes it.
func scannedPDF(t *testing.T) []byte {
	var page bytes.Buffer
	jpeg.Encode(&page, image.NewGray(image.Rect(0, 0, 620, 877)), nil)
	var b bytes.Buffer
	var at []int
	obj := func(body string, stream []byte) {
		at = append(at, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\n", len(at), body)
		if stream != nil {
			b.WriteString("stream\n")
			b.Write(stream)
			b.WriteString("\nendstream\n")
		}
		b.WriteString("endobj\n")
	}
	draw := []byte("q 620 0 0 877 0 0 cm /Im1 Do Q")
	b.WriteString("%PDF-1.4\n")
	obj("<< /Type /Catalog /Pages 2 0 R >>", nil)
	obj("<< /Type /Pages /Kids [3 0 R] /Count 1 >>", nil)
	obj("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 620 877] /Resources << /XObject << /Im1 4 0 R >> >> /Contents 5 0 R >>", nil)
	obj(fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width 620 /Height 877 /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>", page.Len()), page.Bytes())
	obj(fmt.Sprintf("<< /Length %d >>", len(draw)), draw)
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(at)+1)
	for _, o := range at {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(at)+1, xref)
	return b.Bytes()
}

// A letter scanned to a PDF has no words as text; sorting it reads its
// pages as pictures once, keeps their words on the file, names the note
// for what it is, and tags it from those words.
func TestAScanSharedIsReadWhenItIsSorted(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := &readsLetter{}
	a.Chat.Provider = model
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("files", "Scan 12 Nov.pdf")
	fw.Write(scannedPDF(t))
	mw.Close()
	if rec := do(t, h, http.MethodPost, "/share", &buf, mw.FormDataContentType()); rec.Code != http.StatusSeeOther {
		t.Fatalf("shared: %d", rec.Code)
	}
	note := lastNote(t, a)
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Sort", "kind": "classify", "tags": []any{"to do"}})
	a.Automate.Classify(context.Background(), act, "note", note.ID)
	got, _ := a.Store.Get("note", note.ID)
	files, _ := a.Store.List(records.FileType, store.ListOptions{})
	if model.saw != 1 || got.Fields["title"] != "Parking fine from the council" || !strings.Contains(strings.Join(records.StringList(got.Fields["tags"]), ","), "to do") {
		t.Errorf("its page read once, named and tagged from its words: saw %d, %v", model.saw, got.Fields)
	}
	if len(files) != 1 || !strings.Contains(fmt.Sprint(files[0].Fields["text"]), "Pay 35 pounds") || files[0].Fields["status"] != "ready" {
		t.Errorf("its words kept on the file, read: %v", files[0].Fields)
	}
}
