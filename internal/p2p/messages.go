package p2p

import (
	"fmt"
	"math/big"

	"github.com/fxamacker/cbor/v2"
)

const (
	// maxP2PCoinbaseTxSize is the maximum coinbase tx size accepted from P2P peers.
	maxP2PCoinbaseTxSize = 100 * 1024 // 100KB
	// maxP2PMinerAddressLen is the maximum miner address length accepted from P2P peers.
	maxP2PMinerAddressLen = 128
)

const (
	// ProtocolVersion is the current P2P protocol version.
	ProtocolVersion = "1.0.0"

	// ShareTopicName is the GossipSub topic for share propagation.
	ShareTopicName = "/p2pool/shares/" + ProtocolVersion

	// SyncProtocolID is the protocol ID for initial sync.
	// Version 2.0.0: locator-based sync (incompatible with v1 batch sync).
	SyncProtocolID = "/p2pool/sync/2.0.0"
)

// MessageType identifies the type of P2P message.
type MessageType uint8

const (
	MsgTypeShare       MessageType = 1
	MsgTypeTipAnnounce MessageType = 2
	MsgTypeShareReq    MessageType = 3
	MsgTypeShareResp   MessageType = 4
	MsgTypeLocatorReq  MessageType = 5
	MsgTypeLocatorResp MessageType = 6
)

// ShareMsg is a share broadcast via GossipSub.
type ShareMsg struct {
	Type MessageType `cbor:"1,keyasint"`

	// Share header fields (Bitcoin block header)
	Version       int32    `cbor:"2,keyasint"`
	PrevBlockHash [32]byte `cbor:"3,keyasint"`
	MerkleRoot    [32]byte `cbor:"4,keyasint"`
	Timestamp     uint32   `cbor:"5,keyasint"`
	Bits          uint32   `cbor:"6,keyasint"`
	Nonce         uint32   `cbor:"7,keyasint"`

	// Sharechain-specific fields
	ShareVersion    uint32   `cbor:"8,keyasint"`
	PrevShareHash   [32]byte `cbor:"9,keyasint"`
	ShareTargetBits uint32   `cbor:"10,keyasint"` // Compact representation of share target
	MinerAddress    string   `cbor:"11,keyasint"`
	CoinbaseTx      []byte   `cbor:"12,keyasint"`
}

// TipAnnounce announces a node's current chain tip.
type TipAnnounce struct {
	Type      MessageType `cbor:"1,keyasint"`
	TipHash   [32]byte    `cbor:"2,keyasint"`
	Height    int64       `cbor:"3,keyasint"`
	TotalWork []byte      `cbor:"4,keyasint"` // big.Int bytes
}

// ShareRequest requests a batch of shares by hash.
type ShareRequest struct {
	Type      MessageType `cbor:"1,keyasint"`
	StartHash [32]byte    `cbor:"2,keyasint"` // Walk backwards from here
	Count     int         `cbor:"3,keyasint"`
}

// ShareResponse contains a batch of shares.
type ShareResponse struct {
	Type   MessageType `cbor:"1,keyasint"`
	Shares []ShareMsg  `cbor:"2,keyasint"`
}

// Encode serializes a message to CBOR.
func Encode(msg interface{}) ([]byte, error) {
	return cbor.Marshal(msg)
}

// DecodeShareMsg decodes a CBOR-encoded ShareMsg.
func DecodeShareMsg(data []byte) (*ShareMsg, error) {
	var msg ShareMsg
	if err := cbor.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	if len(msg.CoinbaseTx) > maxP2PCoinbaseTxSize {
		return nil, fmt.Errorf("coinbase tx too large: %d bytes", len(msg.CoinbaseTx))
	}
	if len(msg.MinerAddress) > maxP2PMinerAddressLen {
		return nil, fmt.Errorf("miner address too long: %d bytes", len(msg.MinerAddress))
	}
	return &msg, nil
}

// DecodeTipAnnounce decodes a CBOR-encoded TipAnnounce.
func DecodeTipAnnounce(data []byte) (*TipAnnounce, error) {
	var msg TipAnnounce
	if err := cbor.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// DecodeShareRequest decodes a CBOR-encoded ShareRequest.
func DecodeShareRequest(data []byte) (*ShareRequest, error) {
	var msg ShareRequest
	if err := cbor.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// DecodeShareResponse decodes a CBOR-encoded ShareResponse.
func DecodeShareResponse(data []byte) (*ShareResponse, error) {
	var msg ShareResponse
	if err := cbor.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ShareLocatorReq sends exponentially-spaced hashes from the client's chain tip.
type ShareLocatorReq struct {
	Type     MessageType `cbor:"1,keyasint"`
	Locators [][32]byte  `cbor:"2,keyasint"` // tip, tip-1, tip-2, tip-4, tip-8, ..., genesis
	MaxCount int         `cbor:"3,keyasint"` // max shares to return
}

// ShareLocatorResp returns shares from the fork point forward.
type ShareLocatorResp struct {
	Type   MessageType `cbor:"1,keyasint"`
	Shares []ShareMsg  `cbor:"2,keyasint"` // oldest-first (forward order)
	More   bool        `cbor:"3,keyasint"` // true if more shares available
}

// DecodeShareLocatorReq decodes a CBOR-encoded ShareLocatorReq.
func DecodeShareLocatorReq(data []byte) (*ShareLocatorReq, error) {
	var msg ShareLocatorReq
	if err := cbor.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// DecodeShareLocatorResp decodes a CBOR-encoded ShareLocatorResp.
func DecodeShareLocatorResp(data []byte) (*ShareLocatorResp, error) {
	var msg ShareLocatorResp
	if err := cbor.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// BigIntToBytes converts a big.Int to bytes for CBOR encoding.
func BigIntToBytes(n *big.Int) []byte {
	if n == nil {
		return nil
	}
	return n.Bytes()
}

// BytesToBigInt converts bytes back to a big.Int.
func BytesToBigInt(b []byte) *big.Int {
	if len(b) == 0 {
		return new(big.Int)
	}
	return new(big.Int).SetBytes(b)
}
