package soundtrack

import (
	"bufio"
	"errors"
	"io"
)

// EBML, the elements WebM and MKV are made of: each an ID and a size,
// both variable-length numbers, then what is inside.

type ebml struct {
	r   *bufio.Reader
	off int64
}

// vint reads an EBML variable-length number; keep leaves its length marker
// on, as element IDs have it. unknown is a size of all ones.
func (e *ebml) vint(keep bool) (v uint64, unknown bool, err error) {
	b, err := e.r.ReadByte()
	if err != nil {
		return 0, false, err
	}
	e.off++
	n := 1
	for mask := byte(0x80); n <= 8 && b&mask == 0; mask >>= 1 {
		n++
	}
	if n > 8 {
		return 0, false, errors.New("the file's elements are not what they should be")
	}
	v = uint64(b)
	if !keep {
		v &= uint64(0xFF >> n)
	}
	ones := v == uint64(0xFF>>n)
	for i := 1; i < n; i++ {
		c, err := e.r.ReadByte()
		if err != nil {
			return 0, false, err
		}
		e.off++
		v = v<<8 | uint64(c)
		ones = ones && c == 0xFF
	}
	return v, ones && !keep, nil
}

func (e *ebml) skip(n int64) error {
	for n > 0 {
		step := n
		if step > 1<<30 {
			step = 1 << 30
		}
		got, err := e.r.Discard(int(step))
		e.off += int64(got)
		if err != nil {
			return err
		}
		n -= int64(got)
	}
	return nil
}

func (e *ebml) bytes(n int64) ([]byte, error) {
	if n > 16<<20 {
		return nil, errors.New("an element too large to be a table")
	}
	b := make([]byte, n)
	_, err := io.ReadFull(e.r, b)
	e.off += n
	return b, err
}

// elements lists the elements packed in b, which has no unknown sizes.
func elements(b []byte, each func(id uint64, body []byte)) {
	e := &ebml{r: bufio.NewReader(bytesReader(b))}
	for {
		id, _, err := e.vint(true)
		if err != nil {
			return
		}
		size, _, err := e.vint(false)
		if err != nil || int64(size) > int64(len(b)) {
			return
		}
		body, err := e.bytes(int64(size))
		if err != nil {
			return
		}
		each(id, body)
	}
}

type byteReader struct {
	b []byte
	i int
}

func bytesReader(b []byte) *byteReader { return &byteReader{b: b} }
func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func uintOf(b []byte) uint64 {
	var v uint64
	for _, c := range b {
		v = v<<8 | uint64(c)
	}
	return v
}
