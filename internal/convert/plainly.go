package convert

import (
	"bytes"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

// What goes wrong reading a file is said after "Could not read the file:"
// on its page, and to the assistant in the same words. A reader's own
// error ("zip: not a valid zip file", "encrypted PDF: invalid password")
// names the parser's problem, not the person's; these name what happened
// to the file and what they can do (GOV.UK file upload, NN/g error
// messages, WCAG 3.3.3).

// told is a reader's message that is already in a person's words, and is
// said as it is.
type told string

func (t told) Error() string { return string(t) }

// errNoText is a PDF whose pages are pictures.
var errNoText = told("its pages are pictures, such as a scan, with no words in them as text, so there is nothing to read or search. The original opens as it is.")

// ole starts the older Office files, and the newer ones once a password
// locks them: Word, Excel and PowerPoint put a locked file in this box.
var ole = []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}

// plainly says why a file of this ending, holding data, could not be read.
func plainly(ext string, data []byte, err error) error {
	var t told
	switch {
	case errors.As(err, &t):
		return err
	case strings.Contains(err.Error(), "encrypted"), bytes.HasPrefix(data, ole) && (ext == "docx" || ext == "xlsx" || ext == "pptx"):
		return told("it is locked with a password, so its words cannot be read. Open it, save a copy without the password, and add that. The original opens as it is.")
	}
	return told("it seems to be damaged, or is not really a ." + ext + " file, so its words cannot be read. If it opens on this computer, save a fresh copy and add that.")
}

// older is the earlier Office endings, with the newer one Sameway reads
// and the app that saves it.
var older = map[string][2]string{"doc": {"docx", "Word"}, "xls": {"xlsx", "Excel"}, "ppt": {"pptx", "PowerPoint"}}

// noReader says why a file's words cannot be read, in the words its page
// says them after "Could not read the file:". It named workspace.yaml and
// said "no built-in reader for. files", which tells a person nothing they
// can do; what they can do is open the original, which still works.
func noReader(ext string) string {
	if ext == "" {
		return "its name has no ending such as .pdf, so what kind of file it is cannot be told, nor its words read. The original opens as it is."
	}
	if o, ok := older[ext]; ok {
		return "Sameway reads ." + o[0] + " files but not the older ." + ext + ". Open it in " + o[1] + ", save it as ." + o[0] + ", and add that. The original opens as it is."
	}
	return "Sameway cannot read inside ." + ext + " files yet, so their words are not shown or searched. The original opens as it is."
}

// asUTF8 is text as UTF-8 however it was saved: Notepad's "Unicode" is
// UTF-16 with a mark, and a CSV saved by Excel on Windows is often
// Windows-1252, whose é and £ would show as broken characters.
func asUTF8(data []byte) []byte {
	switch {
	case bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}):
		return data[3:]
	case bytes.HasPrefix(data, []byte{0xff, 0xfe}), bytes.HasPrefix(data, []byte{0xfe, 0xff}):
		if out, err := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder().Bytes(data); err == nil {
			return out
		}
	case !utf8.Valid(data):
		if out, err := charmap.Windows1252.NewDecoder().Bytes(data); err == nil {
			return out
		}
	}
	return data
}
