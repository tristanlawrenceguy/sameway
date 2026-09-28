package soundtrack

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// WebM and MKV (Matroska) are EBML elements, one after another: the
// tracks say which is the sound and what it is, and each block in each
// cluster holds a frame of one track at its time. A browser recording
// writes the segment and its clusters without sizes, so the elements are
// read in order, going into those rather than past them.

const (
	idSegment      = 0x18538067
	idInfo         = 0x1549A966
	idScale        = 0x2AD7B1
	idTracks       = 0x1654AE6B
	idTrackEntry   = 0xAE
	idTrackNumber  = 0xD7
	idTrackType    = 0x83
	idCodecID      = 0x86
	idCodecPrivate = 0x63A2
	idCluster      = 0x1F43B675
	idTimestamp    = 0xE7
	idSimpleBlock  = 0xA3
	idBlockGroup   = 0xA0
	idBlock        = 0xA1
)

func readMKV(r io.ReaderAt, size int64) ([]frame, writer, error) {
	e := &ebml{r: bufio.NewReaderSize(io.NewSectionReader(r, 0, size), 1<<16)}
	scale := 1e6 // nanoseconds per timestamp tick
	var track uint64
	var w writer
	var cluster float64
	var frames []frame
	for {
		id, _, err := e.vint(true)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		n, unknown, err := e.vint(false)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, err
		}
		switch id {
		case idSegment, idCluster, idBlockGroup:
			continue // go in
		case idInfo:
			body, err := e.bytes(int64(n))
			if err != nil {
				return nil, nil, err
			}
			elements(body, func(id uint64, b []byte) {
				if id == idScale {
					scale = float64(uintOf(b))
				}
			})
		case idTracks:
			body, err := e.bytes(int64(n))
			if err != nil {
				return nil, nil, err
			}
			track, w, err = soundTrack(body)
			if err != nil {
				return nil, nil, err
			}
		case idTimestamp:
			body, err := e.bytes(int64(n))
			if err != nil {
				return nil, nil, err
			}
			cluster = float64(uintOf(body))
		case idSimpleBlock, idBlock:
			start := e.off
			head, err := e.bytes(min(int64(n), 12))
			if err != nil {
				return nil, nil, err
			}
			if w != nil {
				fs, err := blockFrames(head, start, int64(n), track, (cluster+0)*scale/1e9, scale)
				if err != nil {
					return nil, nil, err
				}
				frames = append(frames, fs...)
			}
			if err := e.skip(int64(n) - int64(len(head))); err != nil {
				if err == io.EOF {
					return finishMKV(frames, w)
				}
				return nil, nil, err
			}
		default:
			if unknown {
				return nil, nil, ErrUnsupported
			}
			if err := e.skip(int64(n)); err != nil {
				if err == io.EOF {
					return finishMKV(frames, w)
				}
				return nil, nil, err
			}
		}
	}
	return finishMKV(frames, w)
}

func finishMKV(frames []frame, w writer) ([]frame, writer, error) {
	if w == nil {
		return nil, nil, errors.New("there is no sound in it")
	}
	return frames, w, nil
}

// soundTrack finds the first sound track and what its frames are.
func soundTrack(tracks []byte) (uint64, writer, error) {
	var number uint64
	var w writer
	err := ErrUnsupported
	found := false
	elements(tracks, func(id uint64, entry []byte) {
		if id != idTrackEntry || found {
			return
		}
		var num, kind uint64
		var codec string
		var private []byte
		elements(entry, func(id uint64, b []byte) {
			switch id {
			case idTrackNumber:
				num = uintOf(b)
			case idTrackType:
				kind = uintOf(b)
			case idCodecID:
				codec = string(b)
			case idCodecPrivate:
				private = b
			}
		})
		if kind != 2 {
			return
		}
		found, number = true, num
		switch codec {
		case "A_OPUS":
			if len(private) >= 19 && string(private[:8]) == "OpusHead" {
				w, err = &oggOpus{head: private}, nil
			}
		case "A_AAC", "A_AAC/MPEG4/LC", "A_AAC/MPEG2/LC":
			var a *adts
			if a, err = parseASC(private); err == nil {
				w = a
			}
		case "A_MPEG/L3":
			w, err = mp3Frames{}, nil
		}
	})
	if !found {
		return 0, nil, errors.New("there is no sound in it")
	}
	return number, w, err
}

// blockFrames reads a block's header (its first bytes, head, at start in
// the file) and lists the frames of the sound track in it.
func blockFrames(head []byte, start, size int64, track uint64, clusterAt, scale float64) ([]frame, error) {
	e := &ebml{r: bufio.NewReader(bytesReader(head))}
	num, _, err := e.vint(false)
	if err != nil || num != track {
		return nil, nil
	}
	if len(head) < int(e.off)+3 {
		return nil, nil
	}
	rel := int16(binary.BigEndian.Uint16(head[e.off:]))
	flags := head[e.off+2]
	at := clusterAt + float64(rel)*scale/1e9
	dataAt := start + e.off + 3
	left := size - (e.off + 3)
	switch (flags >> 1) & 3 {
	case 0:
		return []frame{{off: dataAt, size: int(left), at: math.Max(0, at)}}, nil
	case 2: // fixed-size lacing
		if len(head) < int(e.off)+4 {
			return nil, nil
		}
		count := int64(head[e.off+3]) + 1
		each := (left - 1) / count
		var out []frame
		for i := int64(0); i < count; i++ {
			out = append(out, frame{off: dataAt + 1 + i*each, size: int(each), at: math.Max(0, at)})
		}
		return out, nil
	}
	// Xiph and EBML lacing: rare in files people make, and left to the page.
	return nil, ErrUnsupported
}
