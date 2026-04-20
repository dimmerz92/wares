package sessions

import (
	"bytes"
	"encoding/gob"
	"sync"
)

// Encoder defines a serialiser to encode and decode session data for persistence.
type Encoder interface {
	Marshal(data any) ([]byte, error)
	Unmarshal(encoded []byte, dest any) error
}

// GobEncoder providers a gob based Encoder.
type GobEncoder struct {
	bufPool  sync.Pool
	readPool sync.Pool
}

// NewGobEncoder returns a new instance of a gob encoder.
func NewGobEncoder() Encoder {
	return &GobEncoder{
		bufPool:  sync.Pool{New: func() any { return new(bytes.Buffer) }},
		readPool: sync.Pool{New: func() any { return new(bytes.Reader) }},
	}
}

// Marshal returns the given data as a gob encoded byte slice.
func (e *GobEncoder) Marshal(data any) ([]byte, error) {
	buf := e.bufPool.Get().(*bytes.Buffer)
	defer buf.Reset()

	err := gob.NewEncoder(buf).Encode(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Unmarshal decodes the given gob encoded byte slice into the given destination pointer.
func (e *GobEncoder) Unmarshal(encoded []byte, dest any) error {
	buf := e.readPool.Get().(*bytes.Reader)
	defer buf.Reset(nil)

	buf.Reset(encoded)

	return gob.NewDecoder(buf).Decode(dest)
}
