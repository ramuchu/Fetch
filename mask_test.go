package fetch

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func testmask(t *testing.T, dt dataType, size int, min, max byte) {
	name, method := method(dt)
	b := method(size, min, max)

	var m byte = 0x56

	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	var buf bytes.Buffer
	var msg bytes.Buffer
	wr, ww := NewMaskReader(r2, m), NewMaskWriter(w1, m)
	go func() {
		ww.Write(b)
		w1.Close()
	}()
	c := make(chan struct{})
	go func() {
		buf.ReadFrom(wr)
		close(c)
	}()
	io.Copy(io.MultiWriter(w2, &msg), r1)
	w2.Close()
	<-c

	if !equal(t, b, buf.Bytes(), name, size) {
		c := msg.Bytes()
		d := buf.Bytes()
		fmt.Printf("encoded %d: % X\n", len(c), c)
		fmt.Printf("origin %d: % X\n", len(b), b)
		fmt.Printf("decoded %d: % X\n", len(d), d)
	}
}

func TestMask(t *testing.T) {
	for _, dt := range []dataType{genAcs, genDecs, genRand} {
		for _, size := range []int{0, 1, 2, 10, 256, 1023, 1024, 1025, 1535, 1536, 1537, 2048, 9999} {
			testmask(t, dt, size, 0, 0xff)
			if t.Failed() {
				return
			}
		}
	}
}
