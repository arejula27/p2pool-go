package bitcoin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// BitcoinRPC defines the interface for communicating with bitcoind.
type BitcoinRPC interface {
	GetBlockTemplate(ctx context.Context) (*BlockTemplate, error)
	SubmitBlock(ctx context.Context, blockHex string) error
	GetBlockCount(ctx context.Context) (int64, error)
	GetBestBlockHash(ctx context.Context) (string, error)
}

// RPCClient implements BitcoinRPC using JSON-RPC over HTTP.
type RPCClient struct {
	url      string
	user     string
	password string
	client   *http.Client
	idSeq    atomic.Int64
}

// NewRPCClient creates a new Bitcoin JSON-RPC client.
func NewRPCClient(url, user, password string) *RPCClient {
	return &RPCClient{
		url:      url,
		user:     user,
		password: password,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// call makes a JSON-RPC call and returns the raw result.
func (c *RPCClient) call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	id := c.idSeq.Add(1)

	req := RPCRequest{
		JSONRPC: "1.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(c.user, c.password)

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}

// GetBlockTemplate returns a new block template from bitcoind.
func (c *RPCClient) GetBlockTemplate(ctx context.Context) (*BlockTemplate, error) {
	// getblocktemplate requires a template request parameter
	templateReq := map[string]interface{}{
		"rules": []string{"segwit"},
	}

	result, err := c.call(ctx, "getblocktemplate", templateReq)
	if err != nil {
		return nil, fmt.Errorf("getblocktemplate: %w", err)
	}

	var tmpl BlockTemplate
	if err := json.Unmarshal(result, &tmpl); err != nil {
		return nil, fmt.Errorf("unmarshal block template: %w", err)
	}

	return &tmpl, nil
}

// BlockRejectedError is returned when bitcoind explicitly rejects a block
// (as opposed to a transport/RPC error). Rejected blocks should not be retried.
type BlockRejectedError struct {
	Reason string
}

func (e *BlockRejectedError) Error() string {
	return "block rejected: " + e.Reason
}

// SubmitBlock submits a mined block to the network.
func (c *RPCClient) SubmitBlock(ctx context.Context, blockHex string) error {
	result, err := c.call(ctx, "submitblock", blockHex)
	if err != nil {
		return fmt.Errorf("submitblock: %w", err)
	}

	// submitblock returns null on success, or an error string
	var rejectReason string
	if err := json.Unmarshal(result, &rejectReason); err == nil && rejectReason != "" {
		return &BlockRejectedError{Reason: rejectReason}
	}

	return nil
}

// GetBlockCount returns the current block height.
func (c *RPCClient) GetBlockCount(ctx context.Context) (int64, error) {
	result, err := c.call(ctx, "getblockcount")
	if err != nil {
		return 0, fmt.Errorf("getblockcount: %w", err)
	}

	var height int64
	if err := json.Unmarshal(result, &height); err != nil {
		return 0, fmt.Errorf("unmarshal block count: %w", err)
	}

	return height, nil
}

// GetBestBlockHash returns the hash of the best (tip) block.
func (c *RPCClient) GetBestBlockHash(ctx context.Context) (string, error) {
	result, err := c.call(ctx, "getbestblockhash")
	if err != nil {
		return "", fmt.Errorf("getbestblockhash: %w", err)
	}

	var hash string
	if err := json.Unmarshal(result, &hash); err != nil {
		return "", fmt.Errorf("unmarshal best block hash: %w", err)
	}

	return hash, nil
}
