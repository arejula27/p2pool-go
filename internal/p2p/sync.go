package p2p

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"go.uber.org/zap"
)

const (
	maxSyncBatchSize = 100
	maxSyncMsgSize   = 1024 * 1024 // 1MB
	maxLocatorCount  = 64
	syncStreamTimeout = 30 * time.Second
)

// SyncHandler handles locator-based sync requests from peers.
type SyncHandler func(req *ShareLocatorReq) *ShareLocatorResp

// Syncer handles initial sharechain synchronization.
type Syncer struct {
	host    host.Host
	logger  *zap.Logger
	handler SyncHandler
}

// NewSyncer creates a new sync handler.
func NewSyncer(h host.Host, handler SyncHandler, logger *zap.Logger) *Syncer {
	s := &Syncer{
		host:    h,
		logger:  logger,
		handler: handler,
	}

	h.SetStreamHandler(protocol.ID(SyncProtocolID), s.handleStream)

	return s
}

// handleStream handles incoming sync requests.
func (s *Syncer) handleStream(stream network.Stream) {
	defer stream.Close()

	// Deadline prevents a slow/malicious peer from holding the stream open.
	stream.SetDeadline(time.Now().Add(syncStreamTimeout))

	// Read request (use LimitReader to cap size, ReadAll to get full message)
	data, err := io.ReadAll(io.LimitReader(stream, maxSyncMsgSize))
	if err != nil {
		s.logger.Debug("sync read error", zap.Error(err))
		return
	}

	req, err := DecodeShareLocatorReq(data)
	if err != nil {
		s.logger.Debug("invalid sync request", zap.Error(err))
		return
	}

	if req.MaxCount > maxSyncBatchSize {
		req.MaxCount = maxSyncBatchSize
	}
	if len(req.Locators) > maxLocatorCount {
		req.Locators = req.Locators[:maxLocatorCount]
	}

	// Get response from handler
	resp := s.handler(req)
	if resp == nil {
		resp = &ShareLocatorResp{Type: MsgTypeLocatorResp}
	}

	// Send response
	data, err = Encode(resp)
	if err != nil {
		s.logger.Error("encode sync response", zap.Error(err))
		return
	}

	stream.Write(data)
}

// RequestLocator sends a locator-based sync request to a peer.
func (s *Syncer) RequestLocator(ctx context.Context, peerID peer.ID, locators [][32]byte, maxCount int) (*ShareLocatorResp, error) {
	stream, err := s.host.NewStream(ctx, peerID, protocol.ID(SyncProtocolID))
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}
	defer stream.Close()

	req := &ShareLocatorReq{
		Type:     MsgTypeLocatorReq,
		Locators: locators,
		MaxCount: maxCount,
	}

	data, err := Encode(req)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	if _, err := stream.Write(data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Close write side to signal we're done
	stream.CloseWrite()

	// Read response (use LimitReader to cap size, ReadAll to get full message)
	data, err = io.ReadAll(io.LimitReader(stream, maxSyncMsgSize))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	resp, err := DecodeShareLocatorResp(data)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return resp, nil
}
