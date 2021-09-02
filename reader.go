package timeout

import (
	"io"
	"time"
	"unicode/utf8"
)

type Reader struct {
	buf  []byte
	rd   io.Reader
	l, r int
	ch   chan struct {
		int
		error
	}
}

const defaultBufSize = 4096
const minReadBufferSize = 16

// NewReaderSize creates a Reader from io.Reader while specifying the buffers
// size.
func NewReaderSize(rd io.Reader, size int) *Reader {
	if size < minReadBufferSize {
		size = minReadBufferSize
	}
	reader := &Reader{
		buf: make([]byte, size),
		rd:  rd,
		l:   0,
		r:   0,
		ch: make(chan struct {
			int
			error
		}),
	}
	go reader.readToChannel()
	return reader
}

// NewReader creates a Reader from io.Reader while using the default buffers
// size.
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}

func (rd *Reader) readToChannel() {
	n, err := rd.rd.Read(rd.buf[rd.r:])
	rd.ch <- struct {
		int
		error
	}{n, err}
}

// clean up buffer by shifting data to the left.
func (rd *Reader) shiftBufLeft() {
	if rd.l > 0 {
		rd.r = copy(rd.buf, rd.buf[rd.l:rd.r])
		rd.l = 0
	}
}

func (rd *Reader) isEmpty() bool { return rd.l == rd.r }

func (rd *Reader) isFull() bool { return rd.r == len(rd.buf) }

func (rd *Reader) copyBufTo(p []byte) int {
	count := copy(p, rd.buf[rd.l:rd.r])
	rd.l += count
	return count
}

func (rd *Reader) readIfNotReading() {
	if rd.ch == nil {
		rd.ch = make(chan struct {
			int
			error
		})
		rd.shiftBufLeft()
		go rd.readToChannel()
	}
}

// ReadWithTimeout reads from io.Reader passed until either it get's data or
// there was a timeout. if a read filled the buffer and p is not full, it will
// attempt to read again.
func (rd *Reader) ReadWithTimeout(p []byte, d time.Duration) (n int, err error) {
	// If there is data in the buffer, use that first
	if !rd.isEmpty() {
		count := rd.copyBufTo(p)
		n += count
		p = p[count:]
		// If we filled p, there is nothing more to do
		if len(p) == 0 {
			return
		}
	}

	rd.readIfNotReading()
	c := time.After(d)

	for {
		select {
		case <-c:
			if n == 0 {
				err = ErrTimeout{}
			}
			return
		case s := <-rd.ch:
			rd.r += s.int
			bufFull := rd.isFull()
			count := rd.copyBufTo(p)
			n += count
			p = p[count:]

			if len(p) == 0 || s.error != nil {
				rd.ch = nil
				err = s.error
				return
			}

			// If the buffer was filled, we may have more data available from
			// the rader
			if bufFull {
				rd.shiftBufLeft()
				go rd.readToChannel()
			} else {
				rd.ch = nil
				return
			}
		}
	}
}

// Read reads normally without waiting for a timeout. Good for cleaning the
// buffer and any running read goroutines.
func (rd *Reader)Read(p []byte) (n int, err error) {
	if !rd.isEmpty() {
		count := rd.copyBufTo(p)
		n += count
		p = p[count:]
	}

	if rd.ch != nil {
		s := <- rd.ch
		rd.r += s.int
		bufFull := rd.isFull()
		count := rd.copyBufTo(p)
		n += count
		if s.error != nil {
			err = s.error
			return
		}
		p = p[count:]
		if bufFull {
			count, err = rd.rd.Read(p)
			n += count
		}
		rd.ch = nil
	} else {
		count, e := rd.rd.Read(p)
		n += count
		err = e
		return
	}
	return
}

// ReadByteWithTimeout attempts to read a byte from io.Reader. Will try to read
// until there is data, a read returned an error, or there was a timeout.
func (rd *Reader) ReadByteWithTimeout(d time.Duration) (byte, error) {
	if !rd.isEmpty() {
		c := rd.buf[rd.l]
		rd.l++
		return c, nil
	}

	rd.readIfNotReading()
	c := time.After(d)

	for {
		select {
		case <-c:
			return 0, ErrTimeout{}
		case s := <-rd.ch:
			rd.r += s.int
			if !rd.isEmpty() || s.error != nil {
				rd.ch = nil
				c := rd.buf[rd.l]
				rd.l++
				return c, s.error
			}
			rd.shiftBufLeft()
			go rd.readToChannel()
		}
	}
}

// ReadRuneWithTimeout attempts to read a full rune from io.Reader. Will try to
// read until either there is a full rune in the buffer, a read returned an
// error, or there was a timeout.
func (rd *Reader) ReadRuneWithTimeout(d time.Duration) (r rune, size int, err error) {
	if !rd.isEmpty() && utf8.FullRune(rd.buf[rd.l:rd.r]) {
		r, size = rune(rd.buf[rd.l]), 1
		if r >= utf8.RuneSelf {
			r, size = utf8.DecodeRune(rd.buf[rd.l:rd.r])
		}
		rd.l += size
		return
	}

	rd.readIfNotReading()
	c := time.After(d)

	for {
		select {
		case <-c:
			return 0, 0, ErrTimeout{}
		case s := <-rd.ch:
			rd.r += s.int
			if utf8.FullRune(rd.buf[rd.l:rd.r]) || s.error != nil {
				err = s.error
				if rd.isEmpty() {
					return
				}
				rd.ch = nil
				r, size = rune(rd.buf[rd.l]), 1
				if r >= utf8.RuneSelf {
					r, size = utf8.DecodeRune(rd.buf[rd.l:rd.r])
				}
				rd.l += size
				return
			}
			rd.shiftBufLeft()
			go rd.readToChannel()
		}
	}
}

// ReaderWithTimeout encases a Reader so that it can be used in place of
// io.Reader, io.ByteReader, and io.RuneReader.
type ReaderWithTimeout struct {
	rd *Reader
	d  time.Duration
}

// WithTimeout returns struct that can be used in place of io.Reader,
// io.ByteReader, and io.RuneReader.
func (rd *Reader) WithTimeout(d time.Duration) *ReaderWithTimeout {
	return &ReaderWithTimeout{rd, d}
}

func (rd *ReaderWithTimeout) Read(p []byte) (n int, err error) {
	return rd.rd.ReadWithTimeout(p, rd.d)
}

func (rd *ReaderWithTimeout) ReadByte() (byte, error) {
	return rd.rd.ReadByteWithTimeout(rd.d)
}

func (rd *ReaderWithTimeout) ReadRune() (r rune, size int, err error) {
	return rd.rd.ReadRuneWithTimeout(rd.d)
}
