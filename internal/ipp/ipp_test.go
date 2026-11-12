package ipp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/discover"
)

// A print job is one POST to the printer's path: the attributes every
// printer wants, then the document as it is; the printer's yes is the end
// of it.
func TestAPrintJobCarriesItsAttributesThenTheDocument(t *testing.T) {
	t.Parallel()
	var got Message
	printer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var err error
		if got, err = Decode(body); err != nil || r.URL.Path != "/ipp/print" || r.Header.Get("Content-Type") != "application/ipp" {
			t.Errorf("a print job is an IPP message to the printer's path: %v %s %s", err, r.URL.Path, r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/ipp")
		w.Write(Message{RequestID: got.RequestID}.Encode())
	}))
	defer printer.Close()
	p := Printer{Name: "Kitchen", URL: printer.URL + "/ipp/print"}
	if err := Print(context.Background(), p, strings.NewReader("%PDF-1.4 the letter"), "application/pdf", "Letter"); err != nil {
		t.Fatalf("the printer said yes: %v", err)
	}
	if got.Code != PrintJob {
		t.Errorf("it is a Print-Job: 0x%04x", got.Code)
	}
	want := map[string]string{
		"attributes-charset":          "utf-8",
		"attributes-natural-language": "en",
		"printer-uri":                 "ipp" + strings.TrimPrefix(printer.URL, "http") + "/ipp/print",
		"requesting-user-name":        "Sameway",
		"job-name":                    "Letter",
		"document-format":             "application/pdf",
	}
	for name, v := range want {
		if got.Value(name) != v {
			t.Errorf("%s is %q, want %q", name, got.Value(name), v)
		}
	}
	if len(got.Attrs) == 0 || got.Attrs[0].Name != "attributes-charset" || got.Attrs[0].Group != OperationGroup {
		t.Errorf("the charset comes first, among the operation's attributes: %+v", got.Attrs)
	}
	if string(got.Data) != "%PDF-1.4 the letter" {
		t.Errorf("the document follows the attributes as it is: %q", got.Data)
	}
}

// A printer that will not print says why, and that is said in words a
// person can act on, with its own message when it gives one.
func TestAPrinterThatRefusesSaysWhyInWords(t *testing.T) {
	t.Parallel()
	printer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(Message{Code: 0x040A, RequestID: 1, Attrs: []Attr{
			{Group: OperationGroup, Tag: 0x41, Name: "status-message", Value: "PDF is not supported"},
		}}.Encode())
	}))
	defer printer.Close()
	err := Print(context.Background(), Printer{Name: "Kitchen", URL: printer.URL + "/ipp/print"}, strings.NewReader("x"), "application/pdf", "Letter")
	if err == nil || !strings.Contains(err.Error(), "Kitchen did not print it: it does not print this kind of file (PDF is not supported)") {
		t.Errorf("the refusal in words: %v", err)
	}
	printer.Close()
	if err := Print(context.Background(), Printer{Name: "Kitchen", URL: printer.URL + "/ipp/print"}, strings.NewReader("x"), "application/pdf", "Letter"); err == nil || !strings.Contains(err.Error(), "Kitchen did not answer. It may be switched off") || strings.Contains(err.Error(), "tcp") {
		t.Errorf("a printer that is off says so: %v", err)
	}
}

// A printer's mDNS answer says where jobs go (rp=) and what it takes
// (pdl=); one that says neither is sent to the usual path and let try.
func TestAnAnnouncedPrinterSaysWhereJobsGoAndWhatItTakes(t *testing.T) {
	t.Parallel()
	p, ok := fromService(discover.Service{Type: "_ipp._tcp", Name: "Office laser", Host: "office.local", Addr: "192.168.1.20", Port: 631,
		Text: []string{"txtvers=1", "rp=ipp/print2", "pdl=application/octet-stream,image/urf,image/jpeg"}})
	if !ok || p.URL != "http://192.168.1.20:631/ipp/print2" || p.Name != "Office laser" {
		t.Errorf("sent to its address and its path: %+v", p)
	}
	if !p.Takes("image/jpeg") || p.Takes("application/pdf") {
		t.Errorf("it takes what it lists, and nothing else: %v", p.Formats)
	}
	p, ok = fromService(discover.Service{Type: "_ipp._tcp", Name: "Den", Host: "den.local", Port: 631})
	if !ok || p.URL != "http://den.local:631/ipp/print" || !p.Takes("application/pdf") {
		t.Errorf("one that says nothing is let try at the usual path: %+v", p)
	}
	if _, ok := fromService(discover.Service{Type: "_printer._tcp", Host: "old.local", Port: 515}); ok {
		t.Error("an LPD printer is another protocol")
	}
}
