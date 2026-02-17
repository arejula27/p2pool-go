package p2p

import (
	"context"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// PubSub manages GossipSub for share propagation.
type PubSub struct {
	ps     *pubsub.PubSub
	topic  *pubsub.Topic
	sub    *pubsub.Subscription
	self   peer.ID
	logger *zap.Logger

	peerLimiters   map[peer.ID]*rate.Limiter
	peerLimitersMu sync.Mutex
}

// NewPubSub creates a new GossipSub instance.
func NewPubSub(ctx context.Context, h host.Host, incomingShares chan *ShareMsg, logger *zap.Logger) (*PubSub, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}

	topic, err := ps.Join(ShareTopicName)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	p := &PubSub{
		ps:           ps,
		topic:        topic,
		sub:          sub,
		self:         h.ID(),
		logger:       logger,
		peerLimiters: make(map[peer.ID]*rate.Limiter),
	}

	go p.readLoop(ctx, incomingShares)

	return p, nil
}

// PublishShare publishes a share to the gossipsub network.
func (p *PubSub) PublishShare(share *ShareMsg) error {
	share.Type = MsgTypeShare
	data, err := Encode(share)
	if err != nil {
		return err
	}
	return p.topic.Publish(context.Background(), data)
}

func (p *PubSub) readLoop(ctx context.Context, incomingShares chan *ShareMsg) {
	for {
		msg, err := p.sub.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			p.logger.Error("pubsub read error", zap.Error(err))
			continue
		}

		// Ignore our own messages
		if msg.GetFrom() == p.self {
			continue
		}

		if !p.getPeerLimiter(msg.GetFrom()).Allow() {
			p.logger.Warn("peer rate limited", zap.String("peer", msg.GetFrom().String()))
			continue
		}

		share, err := DecodeShareMsg(msg.Data)
		if err != nil {
			p.logger.Debug("invalid share message", zap.Error(err))
			continue
		}

		select {
		case incomingShares <- share:
		default:
			p.logger.Warn("incoming shares channel full, dropping share")
		}
	}
}

func (p *PubSub) getPeerLimiter(peerID peer.ID) *rate.Limiter {
	p.peerLimitersMu.Lock()
	defer p.peerLimitersMu.Unlock()

	if lim, ok := p.peerLimiters[peerID]; ok {
		return lim
	}

	// Evict a random entry if map is too large
	if len(p.peerLimiters) >= 500 {
		for id := range p.peerLimiters {
			delete(p.peerLimiters, id)
			break
		}
	}

	lim := rate.NewLimiter(10, 20)
	p.peerLimiters[peerID] = lim
	return lim
}
