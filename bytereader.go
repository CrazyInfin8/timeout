package timeout

import (
	"io"
	"time"
)

func init() {
	
}

// ByteReader creates a new 
type ByteReader struct {
	rd io.ByteReader
	ch     chan struct {
		byte
		error
	}
}

// NewByteReader creates a ByteReader from io.ByteReader
func NewByteReader(reader io.ByteReader) *ByteReader {
	return &ByteReader{
		reader, nil,
	}
}

func (br *ByteReader) readByteToChannel(ch chan struct {
	byte
	error
}) {
	b, err := br.rd.ReadByte()
	ch <- struct {
		byte
		error
	}{b, err}
}

// ReadByteWithTimeout attempts to read a byte but will return ErrTimeout if the
// reader takes too long.
func (br *ByteReader) ReadByteWithTimeout(d time.Duration) (b byte, err error) {
	if br.ch == nil {
		br.ch = make(chan struct {
			byte
			error
		})
		go br.readByteToChannel(br.ch)
	}
	select {
	case <-time.After(d):
		return 0, ErrTimeout{}
	case s := <-br.ch:
		b = s.byte
		err = s.error
	}

	close(br.ch)
	br.ch = nil
	return
}

// ReadByte reads normally without waiting for a timeout. Good for cleaning any
// running read goroutines.
func (br *ByteReader) ReadByte() (b byte, err error) {
	if br.ch != nil {
		s := <-br.ch
		close(br.ch)
		br.ch = nil
		return s.byte, s.error
	}
	return br.rd.ReadByte()
}

type ByteReaderWithTimeout struct {
	br *ByteReader
	d time.Duration
}

// WithTimeout returns struct that can be used in place of io.ByteReader while
// still having a timeout
func (br *ByteReader) WithTimeout(d time.Duration) *ByteReaderWithTimeout {
	return &ByteReaderWithTimeout{br, d}
}

func (br *ByteReaderWithTimeout) ReadByte() (b byte, err error) {
	return br.br.ReadByteWithTimeout(br.d)
}