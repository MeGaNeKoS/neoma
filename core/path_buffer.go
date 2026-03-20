package core

import "strconv"

type PathBuffer struct {
	buf []byte
	off int
}

func NewPathBuffer(buf []byte, offset int) *PathBuffer {
	return &PathBuffer{buf: buf, off: offset}
}

func (b *PathBuffer) Push(s string) {
	if len(b.buf) > 0 {
		b.buf = append(b.buf, '.')
	}
	b.buf = append(b.buf, s...)
	b.off = len(b.buf)
}

func (b *PathBuffer) PushIndex(i int) {
	b.buf = append(b.buf, '[')
	b.buf = strconv.AppendInt(b.buf, int64(i), 10)
	b.buf = append(b.buf, ']')
	b.off = len(b.buf)
}

func (b *PathBuffer) Pop() {
	for b.off > 0 {
		b.off--
		if b.buf[b.off] == '.' || b.buf[b.off] == '[' {
			break
		}
	}
	b.buf = b.buf[:b.off]
}

func (b *PathBuffer) With(s string) string {
	b.Push(s)
	tmp := b.String()
	b.Pop()
	return tmp
}

func (b *PathBuffer) WithIndex(i int) string {
	b.PushIndex(i)
	tmp := b.String()
	b.Pop()
	return tmp
}

func (b *PathBuffer) Len() int {
	return b.off
}

func (b *PathBuffer) Bytes() []byte {
	return b.buf[:b.off]
}

func (b *PathBuffer) String() string {
	return string(b.buf[:b.off])
}

func (b *PathBuffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}
