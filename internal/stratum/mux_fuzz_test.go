package stratum

import (
	"bytes"
	"io"
	"testing"
)

// FuzzPrefixConn verifies the key property of prefixConn: reading all bytes
// through any sequence of variably-sized Read calls must produce exactly
// prefix + underlying, in order, with no bytes lost or duplicated.
func FuzzPrefixConn(f *testing.F) {
	// Seed corpus
	f.Add([]byte("G"), []byte("ET / HTTP/1.1\r\n"), 1)
	f.Add([]byte("{"), []byte(`"id":1}`), 4096)
	f.Add([]byte("AB"), []byte("CDEF"), 2)
	f.Add([]byte{}, []byte("hello"), 1)
	f.Add([]byte("x"), []byte{}, 1)
	f.Add([]byte("hello world this is a long prefix"), []byte("and some data"), 3)

	f.Fuzz(func(t *testing.T, prefix, underlying []byte, bufSize int) {
		if bufSize <= 0 {
			bufSize = 1
		}
		if bufSize > 4096 {
			bufSize = 4096
		}

		conn := &prefixConn{
			Conn:   &mockConn{r: bytes.NewReader(underlying)},
			prefix: append([]byte{}, prefix...), // clone to avoid mutation
		}

		want := append(append([]byte{}, prefix...), underlying...)

		// Read using the fuzzed buffer size
		buf := make([]byte, bufSize)
		var got []byte
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				got = append(got, buf[:n]...)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		if !bytes.Equal(got, want) {
			t.Errorf("mismatch: prefix=%d bytes, underlying=%d bytes, bufSize=%d\ngot  %d bytes\nwant %d bytes",
				len(prefix), len(underlying), bufSize, len(got), len(want))
		}
	})
}
