package timeout

import (
	"io"
	"time"
)

// RuneReader reads from an io.RuneReader with a timeout
type RuneReader struct {
	rd io.RuneReader
	ch chan struct {
		rune
		int
		error
	}
}

// NewRuneReader creates a RuneReader from io.RuneReader
func NewRuneReader(reader io.RuneReader) *RuneReader {
	return &RuneReader{
		reader, nil,
	}
}

func (rr *RuneReader) readRuneToChannel(ch chan struct {
	rune
	int
	error
}) {
	r, size, err := rr.rd.ReadRune()
	ch <- struct {
		rune
		int
		error
	}{r, size, err}
}

// ReadRuneWithTimeout attempts to read a rune but will return ErrTimeout if the
// reader takes too long.
func (rr *RuneReader) ReadRuneWithTimeout(d time.Duration) (r rune, size int, err error) {
	if rr.ch == nil {
		rr.ch = make(chan struct {
			rune
			int
			error
		})
		go rr.readRuneToChannel(rr.ch)
	}
	select {
	case <-time.After(d):
		return 0, 0, ErrTimeout{}
	case s := <-rr.ch:
		r = s.rune
		size = s.int
		err = s.error
	}

	rr.ch = nil
	return
}

// ReadRune reads normally without waiting for a timeout. Good for cleaning any
// running read goroutines.
func (rr *RuneReader) ReadRune() (r rune, size int, err error) {
	if rr.ch != nil {
		s := <-rr.ch
		close(rr.ch)
		rr.ch = nil
		return s.rune, s.int, s.error
	}
	return rr.rd.ReadRune()
}

// RuneReaderWithTimeout encases a RuneReader so that it can be used in place of
// io.RuneReader while still having a timeout.
type RuneReaderWithTimeout struct {
	rr *RuneReader
	d  time.Duration
}

// WithTimeout returns struct that can be used in place of io.RuneReader while
// still having a timeout.
func (rr *RuneReader) WithTimeout(d time.Duration) *RuneReaderWithTimeout {
	return &RuneReaderWithTimeout{rr, d}
}

// ReadRune attempts to read a rune from the io.Reader but can stop based on the
// set timeout
func (rr *RuneReaderWithTimeout) ReadRune() (r rune, size int, err error) {
	return rr.rr.ReadRuneWithTimeout(rr.d)
}
