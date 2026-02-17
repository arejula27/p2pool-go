package stratum

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const (
	// writeTimeout is the maximum time to wait for a write to complete.
	writeTimeout = 10 * time.Second

	// maxLineSize is the maximum length of a single JSON-RPC line.
	// Prevents memory exhaustion from a malicious client sending an
	// endless line without a newline terminator.
	maxLineSize = 16 * 1024
)

// Request represents a Stratum JSON-RPC request.
type Request struct {
	ID     interface{}     `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Response represents a Stratum JSON-RPC response.
type Response struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

// Notification represents a server-to-client notification.
type Notification struct {
	ID     interface{}   `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// Codec handles Stratum v1 newline-delimited JSON encoding/decoding.
type Codec struct {
	conn    net.Conn
	scanner *bufio.Scanner
	encoder *json.Encoder
}

// NewCodec creates a new Stratum codec for the given connection.
func NewCodec(conn net.Conn) *Codec {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 4096), maxLineSize)
	return &Codec{
		conn:    conn,
		scanner: scanner,
		encoder: json.NewEncoder(conn),
	}
}

// ReadRequest reads a single Stratum request (newline-delimited JSON).
func (c *Codec) ReadRequest() (*Request, error) {
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return nil, fmt.Errorf("read: %w", err)
		}
		return nil, fmt.Errorf("connection closed")
	}

	var req Request
	if err := json.Unmarshal(c.scanner.Bytes(), &req); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}

	return &req, nil
}

// SendResponse sends a JSON-RPC response.
func (c *Codec) SendResponse(resp *Response) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return c.encoder.Encode(resp)
}

// SendNotification sends a server notification.
func (c *Codec) SendNotification(notif *Notification) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return c.encoder.Encode(notif)
}

// Close closes the underlying connection.
func (c *Codec) Close() error {
	return c.conn.Close()
}
