package convert

import (
	"bytes"
	"image/jpeg"
)

// A scan saved as a PDF (a scanner, a phone's document scanner) is its
// pages as photos with no words as text, so reading it found nothing.
// Most keep each page as a JPEG as it is, and those are found here in the
// file's own bytes, for a model that can see to read: no PDF drawing.

// scanSide is the least a page's long side is, in pixels: smaller
// pictures in a PDF are a logo or a stamp, not a page.
const scanSide = 500

// ScannedPages are the pages of a scanned PDF as JPEGs, in order, up to
// most; none when its pages are drawn some other way.
func ScannedPages(data []byte, most int) [][]byte {
	var pages [][]byte
	for rest := data; len(pages) < most; {
		i := bytes.Index(rest, []byte("stream"))
		if i < 0 {
			break
		}
		rest = rest[i+len("stream"):]
		body := bytes.TrimLeft(rest, "\r\n")
		if !bytes.HasPrefix(body, []byte{0xFF, 0xD8, 0xFF}) {
			continue
		}
		end := bytes.Index(body, []byte("endstream"))
		if end < 0 {
			break
		}
		page := bytes.TrimRight(body[:end], "\r\n")
		if cfg, err := jpeg.DecodeConfig(bytes.NewReader(page)); err == nil && max(cfg.Width, cfg.Height) >= scanSide {
			pages = append(pages, page)
		}
		rest = body[end:]
	}
	return pages
}
