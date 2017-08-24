package fetch

import (
	"io"
	"unicode/utf8"
)

const (
	mask    = 0x55
	bufsize = 1024
)

func bytesize(r rune) int {
	switch i := uint32(r); {
	case i <= 0xff:
		return 1
	case i <= 0xffff:
		return 2
	case i <= 0xffffff:
		return 3
	}
	return 4
}

type encoder struct {
	w   io.Writer
	out [bufsize]byte
	r   [utf8.UTFMax]byte
}

// NewEncoder return a writer that write bytes in a utf-8 compatible way.
func NewEncoder(w io.Writer) io.Writer { return &encoder{w: w} }

// For bytes 0x00-0x7F, it will not change the content.
// For bytes 0x80-0xFF, it may combines the following byte to form 2-3 bytes utf-8 rune.
func (e *encoder) Write(p []byte) (n int, err error) {
	i := 0
	for j := 0; j < len(p); j++ {
		v := p[j]
		if v < utf8.RuneSelf {
			e.out[i] = v ^ mask
			i++
			if i >= bufsize {
				if _, err := e.w.Write(e.out[:i]); err != nil {
					return n, err
				}
				i = 0
			}
		} else {
			v := rune(p[j])
			if (v&0xF8 != 0xD8) && j+1 < len(p) {
				v = (v << 8) | rune(p[j+1])
				j++
				n++
			}
			l := utf8.EncodeRune(e.r[:], v)
			//fmt.Printf("%x % x\n", v, e.r[:l])
			if i+l >= bufsize {
				if _, err := e.w.Write(e.out[:i]); err != nil {
					return n, err
				}
				i = 0
			}
			copy(e.out[i:], e.r[:l])
			i += l
		}
		n++
	}
	if i > 0 {
		if _, err := e.w.Write(e.out[:i]); err != nil {
			return n, err
		}
	}
	return
}

type decoder struct {
	r    io.Reader
	buf  [bufsize]byte
	nbuf int
	rm   [utf8.UTFMax]byte
	nrm  int
}

// NewDecoder returns a reader that convert utf-8 bytes back to bytes
func NewDecoder(r io.Reader) io.Reader { return &decoder{r: r} }

// Read will read the underlying reader and convert from utf-8 bytes to normal bytes
func (d *decoder) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if d.nrm > 0 {
		//fmt.Printf("!!! %d %x\n", d.nrm, d.rm[:d.nrm])
		n = copy(p, d.rm[:d.nrm])
		d.nrm -= n
		if n == len(p) {
			return
		}
	}

	for {
		var nn int
		if d.nbuf > 0 {
			j := 0
			for j < d.nbuf && n < len(p) {
				r, size := utf8.DecodeRune(d.buf[j:d.nbuf])
				//fmt.Println(d.nbuf, j, n, r, size)
				if r == utf8.RuneError && size <= 1 {
					// seems buffer missed sth
					break
				}
				switch i := uint32(r); {
				case i < 128:
					p[n] = byte(r ^ mask)
					n++
				case i < 256:
					p[n] = byte(r)
					n++
				default:
					if i < 0x8000 {
						panic("invalid encoding")
					}
					bs := bytesize(r)
					for i := bs - 1; i >= 0; i-- {
						d.rm[i] = byte(r & 0xff)
						r >>= 8
					}
					cn := copy(p[n:], d.rm[:bs])
					n += cn
					if cn < bs {
						copy(d.rm[:], d.rm[cn:bs])
						d.nrm = bs - cn
					}
				}
				j += size
			}
			if j > 0 {
				//fmt.Printf("j:%d, nbuf:%d n:%d, len(p):%d\n", j, d.nbuf, n, len(p))
				oldn := d.nbuf
				copy(d.buf[:], d.buf[j:d.nbuf])
				d.nbuf -= j
				//fmt.Println("copy", d.nbuf, j)
				if oldn < bufsize || n == len(p) {
					return
				}
			}
		}

		//fmt.Println("err", err)
		if err != nil {
			return n, err
		}

		nn, err = d.r.Read(d.buf[d.nbuf:])
		//fmt.Println(nn, err, d.buf[d.nbuf:nn])
		d.nbuf += nn
	}
}
