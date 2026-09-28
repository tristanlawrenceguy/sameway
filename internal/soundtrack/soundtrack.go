// Package soundtrack copies the sound out of a recording or a video, as it
// is, without decoding it: a gigabyte of video holds a few dozen megabytes
// of sound, and that is all speech-to-text needs. It is copied as chunks of
// about ten minutes, each a whole stream of its own (AAC as ADTS, Opus as
// Ogg, MP3 as its frames), so a browser can decode one chunk at a time and
// no chunk is ever large. The file is read a piece at a time from disk,
// never held whole.
//
// MP4, MOV and M4A (ISO base media) and WebM and MKV (Matroska) are read
// here, with AAC, Opus or MP3 inside, and MP3 files are cut at their frames.
// Anything else answers ErrUnsupported, and is decoded whole the old way.
package soundtrack

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrUnsupported says this file's sound cannot be copied out here.
var ErrUnsupported = errors.New("this kind of sound cannot be copied out")

// Chunk is one piece of the sound: where it starts in the recording, in
// seconds, and the file it was copied to.
type Chunk struct {
	Start float64 `json:"start"`
	Path  string  `json:"-"`
	Ext   string  `json:"ext"`
}

// Every is how much sound goes in one chunk.
var Every = 600.0

// frame is one coded unit of sound: its bytes, where it is in the file,
// and when it starts, in seconds.
type frame struct {
	off  int64
	size int
	at   float64
}

// writer turns frames into one chunk's stream.
type writer interface {
	begin(w io.Writer) error
	frame(w io.Writer, data []byte, at float64) error
	end(w io.Writer) error
	ext() string
}

// Extract copies the sound of the file at src into dir as chunks. dir is
// made and emptied first.
func Extract(src, dir string) ([]Chunk, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	head := make([]byte, 16)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	var frames []frame
	var w writer
	switch {
	case len(head) >= 8 && string(head[4:8]) == "ftyp":
		frames, w, err = readMP4(f, st.Size())
	case len(head) >= 4 && head[0] == 0x1A && head[1] == 0x45 && head[2] == 0xDF && head[3] == 0xA3:
		frames, w, err = readMKV(f, st.Size())
	case len(head) >= 3 && (string(head[:3]) == "ID3" || head[0] == 0xFF && head[1]&0xE0 == 0xE0):
		frames, w, err = readMP3(f, st.Size())
	default:
		return nil, ErrUnsupported
	}
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		return nil, errors.New("there is no sound in it")
	}
	os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return write(f, frames, w, dir)
}

// write copies the frames into chunks, starting a new one every Every
// seconds, and never inside a frame.
func write(r io.ReaderAt, frames []frame, w writer, dir string) ([]Chunk, error) {
	var chunks []Chunk
	var out *os.File
	buf := make([]byte, 0, 1<<16)
	finish := func() error {
		if out == nil {
			return nil
		}
		err := w.end(out)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		out = nil
		return err
	}
	for _, fr := range frames {
		if out == nil || fr.at-chunks[len(chunks)-1].Start >= Every {
			if err := finish(); err != nil {
				return nil, err
			}
			path := filepath.Join(dir, fmt.Sprintf("%d.%s", len(chunks), w.ext()))
			f, err := os.Create(path)
			if err != nil {
				return nil, err
			}
			out = f
			chunks = append(chunks, Chunk{Start: fr.at, Path: path, Ext: w.ext()})
			if err := w.begin(out); err != nil {
				return nil, err
			}
		}
		if cap(buf) < fr.size {
			buf = make([]byte, fr.size)
		}
		data := buf[:fr.size]
		if _, err := r.ReadAt(data, fr.off); err != nil {
			return nil, fmt.Errorf("the sound is cut short: %w", err)
		}
		if err := w.frame(out, data, fr.at-chunks[len(chunks)-1].Start); err != nil {
			return nil, err
		}
	}
	return chunks, finish()
}
