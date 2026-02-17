package stratum

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// mockConn wraps a bytes.Reader as a minimal net.Conn for testing.
type mockConn struct {
	net.Conn // embedded nil — only Read is used
	r        *bytes.Reader
}

func (m *mockConn) Read(p []byte) (int, error)         { return m.r.Read(p) }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }
func (m *mockConn) Write(p []byte) (int, error)        { return len(p), nil }

// --- prefixConn tests ---

func TestPrefixConn_ReadsCorrectly(t *testing.T) {
	underlying := []byte("world")
	prefix := []byte("hello ")

	conn := &prefixConn{
		Conn:   &mockConn{r: bytes.NewReader(underlying)},
		prefix: prefix,
	}

	got, err := io.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}
	want := "hello world"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixConn_SmallBuffer(t *testing.T) {
	// Read one byte at a time through the prefix boundary
	underlying := []byte("BC")
	prefix := []byte("A")

	conn := &prefixConn{
		Conn:   &mockConn{r: bytes.NewReader(underlying)},
		prefix: prefix,
	}

	buf := make([]byte, 1)
	var result []byte
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	if string(result) != "ABC" {
		t.Errorf("got %q, want %q", result, "ABC")
	}
}

func TestPrefixConn_EmptyPrefix(t *testing.T) {
	underlying := []byte("data")
	conn := &prefixConn{
		Conn:   &mockConn{r: bytes.NewReader(underlying)},
		prefix: []byte{},
		read:   true,
	}

	got, err := io.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "data" {
		t.Errorf("got %q, want %q", got, "data")
	}
}

// --- singleConnListener tests ---

func TestSingleConnListener_AcceptOnce(t *testing.T) {
	mc := &mockConn{r: bytes.NewReader(nil)}
	l := &singleConnListener{conn: mc, done: make(chan struct{})}

	// First Accept returns the conn
	c, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	if c != mc {
		t.Error("first Accept should return the original conn")
	}

	// Second Accept should block until Close
	accepted := make(chan error, 1)
	go func() {
		_, err := l.Accept()
		accepted <- err
	}()

	select {
	case <-accepted:
		t.Fatal("second Accept should block")
	case <-time.After(50 * time.Millisecond):
	}

	l.Close()

	select {
	case err := <-accepted:
		if err != net.ErrClosed {
			t.Errorf("got err=%v, want net.ErrClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("second Accept should unblock after Close")
	}
}

func TestSingleConnListener_DoubleClose(t *testing.T) {
	mc := &mockConn{r: bytes.NewReader(nil)}
	l := &singleConnListener{conn: mc, done: make(chan struct{})}

	// Must not panic
	l.Close()
	l.Close()
}

// --- HTTP multiplexing integration test ---

func TestServer_HTTPMultiplexing(t *testing.T) {
	srv := NewServer(1.0, testLogger())

	// Set up a simple HTTP handler
	srv.SetHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))

	if err := srv.Start("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	addr := srv.listener.Addr().String()

	// Test 1: HTTP GET should be routed to the handler
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "OK" {
		t.Errorf("HTTP body = %q, want %q", body, "OK")
	}

	// Test 2: Stratum-like connection (starts with '{') should NOT go to HTTP
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("stratum connect failed: %v", err)
	}
	defer conn.Close()

	// Send a JSON-RPC subscribe — should be handled as stratum
	_, err = conn.Write([]byte(`{"id":1,"method":"mining.subscribe","params":["test/1.0"]}` + "\n"))
	if err != nil {
		t.Fatalf("stratum write failed: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("stratum read failed: %v", err)
	}

	// Should get a valid JSON response, not an HTTP response
	if buf[0] == 'H' { // 'H' as in "HTTP/1.1"
		t.Error("stratum connection was incorrectly routed to HTTP handler")
	}
	if n == 0 {
		t.Error("got empty response from stratum")
	}
}

func TestServer_HTTPWithoutHandler(t *testing.T) {
	// When no HTTP handler is set, non-stratum connections should be
	// treated as stratum (they'll fail to parse, but shouldn't panic).
	srv := NewServer(1.0, testLogger())
	// Intentionally do NOT call SetHTTPHandler

	if err := srv.Start("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	addr := srv.listener.Addr().String()

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer conn.Close()

	// Send "GET / " — not stratum, no HTTP handler set.
	// Should be treated as stratum (prefix '{' check fails but httpHandler is nil).
	conn.Write([]byte("GET / HTTP/1.1\r\nHost: test\r\n\r\n"))
	conn.SetReadDeadline(time.Now().Add(time.Second))

	// Should just close or return an error, not panic
	buf := make([]byte, 1024)
	conn.Read(buf) // don't care about result, just that it didn't crash
}
