package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/security/noise"

	ma "github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

// Node manages the libp2p host and P2P networking.
type Node struct {
	Host   host.Host
	Logger *zap.Logger

	pubsub    *PubSub
	discovery *Discovery
	syncer    *Syncer

	incomingShares chan *ShareMsg
	peerConnected  chan peer.ID
}

// NewNode creates a new libp2p node with GossipSub but does NOT start
// discovery. Call StartDiscovery after registering all stream handlers
// (e.g. InitSyncer) to avoid races where peers connect before handlers
// are ready.
func NewNode(ctx context.Context, listenPort int, dataDir string, logger *zap.Logger) (*Node, error) {
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)

	// Load or create persistent identity (stable peer ID across restarts)
	privKey, err := LoadOrCreateIdentity(dataDir)
	if err != nil {
		return nil, fmt.Errorf("load identity: %w", err)
	}

	cm, err := connmgr.NewConnManager(50, 100, connmgr.WithGracePeriod(time.Minute))
	if err != nil {
		return nil, fmt.Errorf("create connection manager: %w", err)
	}

	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Muxer(yamux.ID, yamux.DefaultTransport),
		libp2p.ConnectionManager(cm),
	)
	if err != nil {
		return nil, fmt.Errorf("create libp2p host: %w", err)
	}

	node := &Node{
		Host:           h,
		Logger:         logger,
		incomingShares: make(chan *ShareMsg, 256),
		peerConnected:  make(chan peer.ID, 16),
	}

	// Register connection notifier to trigger sync on new peers
	h.Network().Notify(&peerNotifiee{peerConnected: node.peerConnected})

	// Setup GossipSub
	node.pubsub, err = NewPubSub(ctx, h, node.incomingShares, logger)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("setup pubsub: %w", err)
	}

	logger.Info("p2p node started",
		zap.String("peer_id", h.ID().String()),
		zap.Int("port", listenPort),
	)

	for _, addr := range h.Addrs() {
		logger.Info("listening on", zap.String("addr", fmt.Sprintf("%s/p2p/%s", addr, h.ID())))
	}

	return node, nil
}

// StartDiscovery begins mDNS and DHT peer discovery. Must be called after
// all stream handlers are registered (InitSyncer, etc.).
func (n *Node) StartDiscovery(ctx context.Context, enableMDNS bool, bootnodes []string) error {
	var err error
	n.discovery, err = NewDiscovery(ctx, n.Host, enableMDNS, bootnodes, n.Logger)
	if err != nil {
		return fmt.Errorf("setup discovery: %w", err)
	}
	return nil
}

// IncomingShares returns the channel of shares received from peers.
func (n *Node) IncomingShares() <-chan *ShareMsg {
	return n.incomingShares
}

// BroadcastShare publishes a share to the network.
func (n *Node) BroadcastShare(share *ShareMsg) error {
	return n.pubsub.PublishShare(share)
}

// PeerCount returns the number of connected peers.
func (n *Node) PeerCount() int {
	return len(n.Host.Network().Peers())
}

// ConnectedPeers returns info about connected peers.
func (n *Node) ConnectedPeers() []peer.ID {
	return n.Host.Network().Peers()
}

// InitSyncer creates the Syncer and registers the stream handler.
// Must be called after the sharechain is ready.
func (n *Node) InitSyncer(handler SyncHandler) {
	n.syncer = NewSyncer(n.Host, handler, n.Logger)
}

// PeerConnected returns a channel that receives peer IDs when new peers connect.
func (n *Node) PeerConnected() <-chan peer.ID {
	return n.peerConnected
}

// Syncer returns the sync protocol handler.
func (n *Node) Syncer() *Syncer {
	return n.syncer
}

// Close shuts down the node.
func (n *Node) Close() error {
	return n.Host.Close()
}

// peerNotifiee implements network.Notifiee to detect new peer connections.
type peerNotifiee struct {
	peerConnected chan peer.ID
}

func (pn *peerNotifiee) Connected(_ network.Network, conn network.Conn) {
	// Non-blocking send; drop if channel is full (sync will happen on next connect)
	select {
	case pn.peerConnected <- conn.RemotePeer():
	default:
	}
}

func (pn *peerNotifiee) Disconnected(network.Network, network.Conn) {}
func (pn *peerNotifiee) Listen(network.Network, ma.Multiaddr)      {}
func (pn *peerNotifiee) ListenClose(network.Network, ma.Multiaddr) {}
