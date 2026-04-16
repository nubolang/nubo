package plug

import (
	"encoding/binary"
	"io"
	"sync"

	"github.com/ugorji/go/codec"
)

var mh = &codec.MsgpackHandle{}

// frame is the wire-level message for both requests and responses.
type frame struct {
	ID     uint32 `codec:"id"`
	Method string `codec:"method,omitempty"`
	Params []byte `codec:"params,omitempty"`
	Result []byte `codec:"result,omitempty"`
	Err    string `codec:"err,omitempty"`
}

// frameWriter serialises frames safely from multiple goroutines.
type frameWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (fw *frameWriter) write(f *frame) error {
	data, err := marshalFrame(f)
	if err != nil {
		return err
	}
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, uint32(len(data)))

	fw.mu.Lock()
	defer fw.mu.Unlock()
	if _, err = fw.w.Write(hdr); err != nil {
		return err
	}
	_, err = fw.w.Write(data)
	return err
}

func readFrame(r io.Reader) (*frame, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr)
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	var f frame
	dec := codec.NewDecoderBytes(buf, mh)
	return &f, dec.Decode(&f)
}

func marshalFrame(f *frame) ([]byte, error) {
	var buf []byte
	enc := codec.NewEncoderBytes(&buf, mh)
	return buf, enc.Encode(f)
}

// Marshal encodes v to msgpack bytes.
func Marshal(v any) ([]byte, error) {
	var buf []byte
	enc := codec.NewEncoderBytes(&buf, mh)
	return buf, enc.Encode(v)
}

// Unmarshal decodes msgpack bytes into v.
func Unmarshal(data []byte, v any) error {
	dec := codec.NewDecoderBytes(data, mh)
	return dec.Decode(v)
}
