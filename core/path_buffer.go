package core

import "strconv"

// PathBuffer is an efficient builder for dot-separated validation error paths
// such as "body.items[3].name". It supports push/pop semantics so that path
// segments can be appended while descending into nested structures and removed
// when returning.
type PathBuffer struct {
	buf []byte
	off int
}

// NewPathBuffer returns a PathBuffer initialized with the given backing byte
// slice and offset.
func NewPathBuffer(buf []byte, offset int) *PathBuffer {
	return &PathBuffer{buf: buf, off: offset}
}

// Push appends a dot-separated property name to the path (e.g. "name").
func (b *PathBuffer) Push(s string) {
	if len(b.buf) > 0 {
		b.buf = append(b.buf, '.')
	}
	b.buf = append(b.buf, s...)
	b.off = len(b.buf)
}

// PushIndex appends a bracketed array index to the path (e.g. "[3]").
func (b *PathBuffer) PushIndex(i int) {
	b.buf = append(b.buf, '[')
	b.buf = strconv.AppendInt(b.buf, int64(i), 10)
	b.buf = append(b.buf, ']')
	b.off = len(b.buf)
}

// Pop removes the last segment (property name or array index) from the path.
func (b *PathBuffer) Pop() {
	for b.off > 0 {
		b.off--
		if b.buf[b.off] == '.' || b.buf[b.off] == '[' {
			break
		}
	}
	b.buf = b.buf[:b.off]
}

// With temporarily pushes a property name, snapshots the path as a string,
// then pops it. This is a convenience for one-off path construction.
func (b *PathBuffer) With(s string) string {
	b.Push(s)
	tmp := b.String()
	b.Pop()
	return tmp
}

// WithIndex temporarily pushes an array index, snapshots the path as a
// string, then pops it. This is a convenience for one-off path construction.
func (b *PathBuffer) WithIndex(i int) string {
	b.PushIndex(i)
	tmp := b.String()
	b.Pop()
	return tmp
}

// Len returns the current length of the path in bytes.
func (b *PathBuffer) Len() int {
	return b.off
}

// Bytes returns the current path as a byte slice.
func (b *PathBuffer) Bytes() []byte {
	return b.buf[:b.off]
}

// String returns the current path as a string.
func (b *PathBuffer) String() string {
	return string(b.buf[:b.off])
}

// Reset clears the path buffer, allowing it to be reused.
func (b *PathBuffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}
