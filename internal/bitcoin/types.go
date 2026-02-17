package bitcoin

import (
	"encoding/json"
	"fmt"
)

// BlockTemplate represents the response from getblocktemplate RPC.
type BlockTemplate struct {
	Version                  int32                 `json:"version"`
	PreviousBlockHash        string                `json:"previousblockhash"`
	Transactions             []TemplateTransaction `json:"transactions"`
	CoinbaseAux              *CoinbaseAux          `json:"coinbaseaux"`
	CoinbaseValue            int64                 `json:"coinbasevalue"`
	Target                   string                `json:"target"`
	MinTime                  int64                 `json:"mintime"`
	Mutable                  []string              `json:"mutable"`
	NonceRange               string                `json:"noncerange"`
	SigOpLimit               int                   `json:"sigoplimit"`
	SizeLimit                int                   `json:"sizelimit"`
	WeightLimit              int                   `json:"weightlimit"`
	CurTime                  int64                 `json:"curtime"`
	Bits                     string                `json:"bits"`
	Height                   int64                 `json:"height"`
	DefaultWitnessCommitment string                `json:"default_witness_commitment"`
}

// TemplateTransaction represents a transaction in a block template.
type TemplateTransaction struct {
	Data   string `json:"data"`
	TxID   string `json:"txid"`
	Hash   string `json:"hash"`
	Fee    int64  `json:"fee"`
	SigOps int    `json:"sigops"`
	Weight int    `json:"weight"`
}

// CoinbaseAux contains auxiliary data for the coinbase.
type CoinbaseAux struct {
	Flags string `json:"flags"`
}

// BlockInfo represents basic block information from getblock.
type BlockInfo struct {
	Hash          string  `json:"hash"`
	Height        int64   `json:"height"`
	Version       int32   `json:"version"`
	PreviousHash  string  `json:"previousblockhash"`
	Time          int64   `json:"time"`
	Bits          string  `json:"bits"`
	Difficulty    float64 `json:"difficulty"`
	Confirmations int64   `json:"confirmations"`
}

// RPCRequest represents a JSON-RPC request.
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params,omitempty"`
}

// RPCResponse represents a JSON-RPC response.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}
