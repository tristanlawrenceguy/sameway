package soundtrack

import (
	"encoding/binary"
)

// sampleTable lists every frame of the track: its offset from the chunk
// offsets and sizes, its time from the durations.
func sampleTable(stbl []byte, scale float64) ([]frame, error) {
	stsz, stsc, stts := child(stbl, "stsz"), child(stbl, "stsc"), child(stbl, "stts")
	var offsets []int64
	if co := child(stbl, "stco"); len(co) >= 8 {
		n := int(binary.BigEndian.Uint32(co[4:]))
		for i := 0; i < n && 8+4*i+4 <= len(co); i++ {
			offsets = append(offsets, int64(binary.BigEndian.Uint32(co[8+4*i:])))
		}
	} else if co := child(stbl, "co64"); len(co) >= 8 {
		n := int(binary.BigEndian.Uint32(co[4:]))
		for i := 0; i < n && 8+8*i+8 <= len(co); i++ {
			offsets = append(offsets, int64(binary.BigEndian.Uint64(co[8+8*i:])))
		}
	}
	if len(stsz) < 12 || len(stsc) < 8 || len(stts) < 8 || len(offsets) == 0 || scale == 0 {
		return nil, ErrUnsupported
	}
	fixed, count := int(binary.BigEndian.Uint32(stsz[4:])), int(binary.BigEndian.Uint32(stsz[8:]))
	sizeOf := func(i int) int {
		if fixed != 0 {
			return fixed
		}
		if 12+4*i+4 > len(stsz) {
			return 0
		}
		return int(binary.BigEndian.Uint32(stsz[12+4*i:]))
	}
	// Durations, run by run.
	var deltas []uint32
	var runs []uint32
	for i, n := 0, int(binary.BigEndian.Uint32(stts[4:])); i < n && 8+8*i+8 <= len(stts); i++ {
		runs = append(runs, binary.BigEndian.Uint32(stts[8+8*i:]))
		deltas = append(deltas, binary.BigEndian.Uint32(stts[12+8*i:]))
	}
	type run struct{ first, per int }
	var chunks []run
	for i, n := 0, int(binary.BigEndian.Uint32(stsc[4:])); i < n && 8+12*i+12 <= len(stsc); i++ {
		chunks = append(chunks, run{int(binary.BigEndian.Uint32(stsc[8+12*i:])), int(binary.BigEndian.Uint32(stsc[12+12*i:]))})
	}
	frames := make([]frame, 0, count)
	sample, ri, left := 0, 0, uint32(0)
	if len(runs) > 0 {
		left = runs[0]
	}
	var ticks uint64
	for c := range offsets {
		per := 0
		for _, rn := range chunks {
			if rn.first <= c+1 {
				per = rn.per
			}
		}
		off := offsets[c]
		for k := 0; k < per && sample < count; k++ {
			size := sizeOf(sample)
			frames = append(frames, frame{off: off, size: size, at: float64(ticks) / scale})
			off += int64(size)
			for left == 0 && ri+1 < len(runs) {
				ri++
				left = runs[ri]
			}
			if ri < len(deltas) {
				ticks += uint64(deltas[ri])
			}
			if left > 0 {
				left--
			}
			sample++
		}
	}
	return frames, nil
}
