package fetch

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"testing"
)

type dataType int

const (
	genAcs = iota
	genDecs
	genRand
)

type genData func(n int, min, max byte) []byte

func dataAcs(n int, min, max byte) []byte {
	b := make([]byte, n)
	x := min
	for i := range b {
		b[i] = x
		if x+1 >= max {
			x = min
		} else {
			x++
		}
	}
	return b
}

func dataDecs(n int, min, max byte) []byte {
	b := make([]byte, n)
	x := max
	for i := range b {
		b[i] = x
		if x-1 <= min {
			x = max
		} else {
			x--
		}
	}
	return b
}

func dataRand(n int, min, max byte) []byte {
	b := make([]byte, n)
	rand.Read(b)
	r := int(max - min + 1)
	for i, v := range b {
		if v < min || v > max {
			b[i] = byte(rand.Intn(r)) + min
		}
	}
	return b
}

func method(t dataType) (name string, fn genData) {
	switch t {
	case genAcs:
		return "dataAcs", dataAcs
	case genDecs:
		return "dataDecs", dataDecs
	case genRand:
		return "dataRand", dataRand
	default:
		return "unknown", dataAcs
	}
}

func equal(t *testing.T, a, b []byte, name string, size int) bool {
	if len(a) != len(b) {
		t.Errorf("%s[%d] Wrong count: %d %d", name, size, len(a), len(b))
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("%s[%d] Wrong value: at %d: expect %X see %X", name, size, i, a[i], b[i])
			return false
		}
	}
	return true
}

func test(t *testing.T, dt dataType, size int, min, max byte) {
	name, method := method(dt)
	b := method(size, min, max)

	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	var buf bytes.Buffer
	var msg bytes.Buffer
	wr, ww := NewDecoder(r2), NewEncoder(w1)
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

func TestEncode(t *testing.T) {
	for _, dt := range []dataType{genAcs, genDecs, genRand} {
		for _, size := range []int{0, 1, 2, 10, 256, 1023, 1024, 1025, 1535, 1536, 1537, 2048, 9999} {
			test(t, dt, size, 0, 0xff)
			if t.Failed() {
				return
			}
		}
	}
}

func TestHighbit(t *testing.T) {
	for _, dt := range []dataType{genAcs, genDecs, genRand} {
		for _, size := range []int{0, 1, 2, 10, 256, 1023, 1024, 1025, 1535, 1536, 1537, 2048, 9999} {
			test(t, dt, size, 0x80, 0xff)
			if t.Failed() {
				return
			}
		}
	}
}
