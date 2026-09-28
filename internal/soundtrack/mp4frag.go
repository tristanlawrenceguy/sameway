package soundtrack

import (
	"encoding/binary"
	"errors"
	"io"
)

// A fragmented MP4, as a browser or a phone writes while it records, keeps
// its table in pieces: each moof box says where its samples are in the
// mdat that follows it and how long each lasts, from defaults in the moov.

// trackID is a track's number, from its tkhd box.
func trackID(trak []byte) uint32 {
	tkhd := child(trak, "tkhd")
	if len(tkhd) >= 24 && tkhd[0] == 1 {
		return binary.BigEndian.Uint32(tkhd[20:])
	}
	if len(tkhd) >= 16 {
		return binary.BigEndian.Uint32(tkhd[12:])
	}
	return 0
}

// fragments lists the frames of track id across every moof in the file.
func fragments(r io.ReaderAt, size int64, moov []byte, id uint32, scale float64) ([]frame, error) {
	var defDur, defSize uint32
	for _, bx := range boxes(child(moov, "mvex")) {
		if bx.typ == "trex" && len(bx.body) >= 24 && binary.BigEndian.Uint32(bx.body[4:]) == id {
			defDur, defSize = binary.BigEndian.Uint32(bx.body[12:]), binary.BigEndian.Uint32(bx.body[16:])
		}
	}
	var frames []frame
	var off int64
	h := make([]byte, 16)
	for off+8 <= size {
		if _, err := r.ReadAt(h[:8], off); err != nil {
			return nil, err
		}
		n, head := int64(binary.BigEndian.Uint32(h)), int64(8)
		if n == 1 {
			if _, err := r.ReadAt(h[8:16], off+8); err != nil {
				return nil, err
			}
			n, head = int64(binary.BigEndian.Uint64(h[8:16])), 16
		} else if n == 0 {
			n = size - off
		}
		if n < head || off+n > size {
			break // a recording cut off mid-box keeps what came before
		}
		if string(h[4:8]) == "moof" {
			if n > 64<<20 {
				return nil, ErrUnsupported
			}
			moof := make([]byte, n-head)
			if _, err := r.ReadAt(moof, off+head); err != nil {
				return nil, err
			}
			fs, err := trafFrames(moof, off, id, defDur, defSize, scale)
			if err != nil {
				return nil, err
			}
			frames = append(frames, fs...)
		}
		off += n
	}
	if len(frames) == 0 {
		return nil, errors.New("there is no sound in it")
	}
	return frames, nil
}

// trafFrames reads one moof's run of the track's samples; moofAt is where
// the moof starts, which offsets count from.
func trafFrames(moof []byte, moofAt int64, id, defDur, defSize uint32, scale float64) ([]frame, error) {
	var out []frame
	for _, traf := range boxes(moof) {
		if traf.typ != "traf" {
			continue
		}
		tfhd := child(traf.body, "tfhd")
		if len(tfhd) < 8 || binary.BigEndian.Uint32(tfhd[4:]) != id {
			continue
		}
		flags := uint32(tfhd[1])<<16 | uint32(tfhd[2])<<8 | uint32(tfhd[3])
		base, p := moofAt, 8
		if flags&0x1 != 0 && len(tfhd) >= p+8 {
			base = int64(binary.BigEndian.Uint64(tfhd[p:]))
			p += 8
		}
		if flags&0x2 != 0 {
			p += 4
		}
		dur, sz := defDur, defSize
		if flags&0x8 != 0 && len(tfhd) >= p+4 {
			dur = binary.BigEndian.Uint32(tfhd[p:])
			p += 4
		}
		if flags&0x10 != 0 && len(tfhd) >= p+4 {
			sz = binary.BigEndian.Uint32(tfhd[p:])
		}
		var t uint64
		if tfdt := child(traf.body, "tfdt"); len(tfdt) >= 8 {
			if tfdt[0] == 1 && len(tfdt) >= 12 {
				t = binary.BigEndian.Uint64(tfdt[4:])
			} else {
				t = uint64(binary.BigEndian.Uint32(tfdt[4:]))
			}
		}
		for _, trun := range boxes(traf.body) {
			if trun.typ != "trun" || len(trun.body) < 8 {
				continue
			}
			b := trun.body
			tf := uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
			count := int(binary.BigEndian.Uint32(b[4:]))
			q := 8
			at := base
			if tf&0x1 != 0 && len(b) >= q+4 {
				at = base + int64(int32(binary.BigEndian.Uint32(b[q:])))
				q += 4
			}
			if tf&0x4 != 0 {
				q += 4
			}
			for i := 0; i < count; i++ {
				d, s := dur, sz
				if tf&0x100 != 0 && len(b) >= q+4 {
					d = binary.BigEndian.Uint32(b[q:])
					q += 4
				}
				if tf&0x200 != 0 && len(b) >= q+4 {
					s = binary.BigEndian.Uint32(b[q:])
					q += 4
				}
				if tf&0x400 != 0 {
					q += 4
				}
				if tf&0x800 != 0 {
					q += 4
				}
				out = append(out, frame{off: at, size: int(s), at: float64(t) / scale})
				at += int64(s)
				t += uint64(d)
			}
		}
	}
	return out, nil
}
