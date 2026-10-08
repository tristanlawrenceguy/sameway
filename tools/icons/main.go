// Command icons draws Sameway's icon and writes it in every form a system
// asks for, into design/brand: icon.svg for the browser tab, the app's
// sizes for installing it (icon-192, icon-512, and square ones), icon.png for
// Linux and the Windows resource the release builds, icon.ico for
// Windows, icon.icns for the Mac app. The icon is shapes, not a picture,
// so it is drawn here with the standard library and every size is sharp:
// a rounded square in the accent colour and two strokes going the same way.
//
//	go run ./tools/icons
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// The icon on a 64-unit square.
var (
	accent  = color.NRGBA{0x1a, 0x45, 0xa8, 0xff} // --sw-color-accent, light
	radius  = 14.0
	strokes = [][4]float64{{15, 46, 31, 18}, {33, 46, 49, 18}}
	width   = 8.0
)

const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="14" fill="#1a45a8"/><path d="M15 46 31 18M33 46 49 18" stroke="#fff" stroke-width="8" stroke-linecap="round"/></svg>
`

func main() {
	out := filepath.Join("design", "brand")
	must(os.MkdirAll(out, 0o755))
	must(os.WriteFile(filepath.Join(out, "icon.svg"), []byte(svg), 0o644))
	must(os.WriteFile(filepath.Join(out, "icon.png"), encode(draw(256)), 0o644))
	must(os.WriteFile(filepath.Join(out, "icon.ico"), ico(16, 24, 32, 48, 64, 256), 0o644))
	// For installing it as an app: the sizes a browser asks for, and a
	// square without corners for a phone's home screen and for masking,
	// which round it themselves (the strokes keep well inside the safe zone).
	must(os.WriteFile(filepath.Join(out, "icon-192.png"), encode(draw(192)), 0o644))
	must(os.WriteFile(filepath.Join(out, "icon-512.png"), encode(draw(512)), 0o644))
	radius = 0
	must(os.WriteFile(filepath.Join(out, "icon-square-180.png"), encode(draw(180)), 0o644))
	must(os.WriteFile(filepath.Join(out, "icon-square-512.png"), encode(draw(512)), 0o644))
	must(os.WriteFile(filepath.Join(out, "icon.icns"), icns(), 0o644))
	fmt.Println("wrote", out)
}

// draw renders the icon at n pixels square, each pixel the average of a
// 4 by 4 grid of samples so its edges are smooth.
func draw(n int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	const ss = 4
	scale := 64 / float64(n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := (float64(x) + (float64(sx)+0.5)/ss) * scale
					py := (float64(y) + (float64(sy)+0.5)/ss) * scale
					if !inRounded(px, py) {
						continue
					}
					c := accent
					for _, s := range strokes {
						if segDist(px, py, s) <= width/2 {
							c = color.NRGBA{0xff, 0xff, 0xff, 0xff}
						}
					}
					r += float64(c.R)
					g += float64(c.G)
					b += float64(c.B)
					a += 255
				}
			}
			if a == 0 {
				continue
			}
			// Colour is averaged over the samples that are inside, coverage
			// over all of them.
			k := a / 255
			img.SetNRGBA(x, y, color.NRGBA{uint8(r / k), uint8(g / k), uint8(b / k), uint8(a / (ss * ss))})
		}
	}
	return img
}

func inRounded(x, y float64) bool {
	if radius == 0 {
		return x >= 0 && y >= 0 && x <= 64 && y <= 64
	}
	cx := math.Max(radius, math.Min(64-radius, x))
	cy := math.Max(radius, math.Min(64-radius, y))
	return x >= 0 && y >= 0 && x <= 64 && y <= 64 && math.Hypot(x-cx, y-cy) <= radius
}

func segDist(x, y float64, s [4]float64) float64 {
	dx, dy := s[2]-s[0], s[3]-s[1]
	t := ((x-s[0])*dx + (y-s[1])*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(x-(s[0]+t*dx), y-(s[1]+t*dy))
}

func encode(img image.Image) []byte {
	var b bytes.Buffer
	must(png.Encode(&b, img))
	return b.Bytes()
}

// ico is a Windows icon of PNG images, which every Windows since Vista reads.
func ico(sizes ...int) []byte {
	var head, body bytes.Buffer
	binary.Write(&head, binary.LittleEndian, []uint16{0, 1, uint16(len(sizes))})
	offset := 6 + 16*len(sizes)
	for _, n := range sizes {
		data := encode(draw(n))
		dim := byte(n)
		if n >= 256 {
			dim = 0
		}
		head.Write([]byte{dim, dim, 0, 0})
		binary.Write(&head, binary.LittleEndian, []uint16{1, 32})
		binary.Write(&head, binary.LittleEndian, []uint32{uint32(len(data)), uint32(offset)})
		offset += len(data)
		body.Write(data)
	}
	return append(head.Bytes(), body.Bytes()...)
}

// icns is a Mac icon of PNG images, by the type each size is filed under.
func icns() []byte {
	var body bytes.Buffer
	for _, e := range []struct {
		kind string
		n    int
	}{{"icp4", 16}, {"icp5", 32}, {"icp6", 64}, {"ic07", 128}, {"ic08", 256}, {"ic09", 512}, {"ic10", 1024}} {
		data := encode(draw(e.n))
		body.WriteString(e.kind)
		binary.Write(&body, binary.BigEndian, uint32(8+len(data)))
		body.Write(data)
	}
	var out bytes.Buffer
	out.WriteString("icns")
	binary.Write(&out, binary.BigEndian, uint32(8+body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "icons:", err)
		os.Exit(1)
	}
}
