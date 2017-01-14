package peerheap

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
)

const (
	_retryingPenalty         = 1000
	_connectingPenalty       = 50
	_otherUnconnectedPenalty = 0
)

type List struct {
	sync.Mutex

	transport peer.Transport

	byScore      peerHeap
	byIdentifier map[peer.Identifier]*peerScore

	added int // incrementing counter for when a peer was pushed.

	peerAvailableEvent chan struct{}
}

func (pl *List) IsRunning() bool {
	return true
}

func (pl *List) Start() error {
	return nil
}

func (pl *List) Stop() error {
	return nil
}

func New(transport peer.Transport) *List {
	return &List{
		transport:          transport,
		byIdentifier:       make(map[peer.Identifier]*peerScore),
		peerAvailableEvent: make(chan struct{}, 1),
	}
}

func (pl *List) Update(updates peer.ListUpdates) error {
	pl.Lock()
	defer pl.Unlock()

	add := updates.Additions
	remove := updates.Removals

	for _, pid := range remove {
		pl.removePeer(pid)
	}
	for _, pid := range add {
		pl.addPeer(pid)
	}

	// TODO collect errors
	return nil
}

func (pl *List) addPeer(pid peer.Identifier) error {
	if _, ok := pl.byIdentifier[pid]; ok {
		return fmt.Errorf("tried to add existing peer: %q", pid.Identifier())
	}

	ps := &peerScore{list: pl}
	peer, err := pl.transport.RetainPeer(pid, ps)
	if err != nil {
		return err
	}
	ps.peer = peer
	ps.id = pid
	ps.score = scorePeer(peer)
	ps.boundFinish = ps.finish
	pl.byIdentifier[pid] = ps
	pl.byScore.pushPeer(ps)
	pl.internalNotifyStatusChanged(ps)
	return nil
}

func (pl *List) removePeer(pid peer.Identifier) error {
	ps, ok := pl.byIdentifier[pid]
	if !ok {
		return fmt.Errorf("tried to remove missing peer: %q", pid)
	}

	if err := pl.byScore.validate(ps); err != nil {
		return err
	}

	err := pl.transport.ReleasePeer(ps.id, ps)
	delete(pl.byIdentifier, ps.peer)
	pl.byScore.delete(ps.idx)
	return err
}

func (pl *List) Choose(ctx context.Context, _ *transport.Request) (peer.Peer, func(error), error) {
	for {
		if ps, ok := pl.get(); ok {
			pl.notifyPeerAvailable()
			return ps.peer, ps.boundFinish, nil
		}

		if err := pl.waitForPeerAvailableEvent(ctx); err != nil {
			return nil, nil, err
		}
	}
}

func (pl *List) get() (*peerScore, bool) {
	pl.Lock()
	defer pl.Unlock()

	ps, ok := pl.byScore.popPeer()
	if !ok {
		return nil, false
	}

	// Note: We push the peer back to reset the "last" timestamp.
	// This gives us round-robin behaviour.
	pl.byScore.pushPeer(ps)

	return ps, ps.status.ConnectionStatus == peer.Available
}

// waitForPeerAvailableEvent waits until a peer is added to the peer list or the
// given context finishes.
// Must NOT be run in a mutex.Lock()
func (pl *List) waitForPeerAvailableEvent(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return peer.ErrChooseContextHasNoDeadline("PeerHeap")
	}

	select {
	case <-pl.peerAvailableEvent:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// notifyPeerAvailable writes to a channel indicating that a Peer is currently
// available for requests
func (pl *List) notifyPeerAvailable() {
	select {
	case pl.peerAvailableEvent <- struct{}{}:
	default:
	}
}

func (pl *List) NotifyStatusChanged(pid peer.Identifier) {
	pl.Lock()
	ps := pl.byIdentifier[pid]
	pl.Unlock()
	ps.NotifyStatusChanged(pid)
}

func (pl *List) notifyStatusChanged(ps *peerScore) {
	pl.Lock()
	p := ps.peer
	ps.status = p.Status()
	ps.score = scorePeer(p)
	pl.byScore.update(ps.idx)
	pl.Unlock()

	if p.Status().ConnectionStatus == peer.Available {
		pl.notifyPeerAvailable()
	}
}

func (pl *List) internalNotifyStatusChanged(ps *peerScore) {
	p := ps.peer
	ps.status = p.Status()
	ps.score = scorePeer(p)
	pl.byScore.update(ps.idx)

	if p.Status().ConnectionStatus == peer.Available {
		pl.notifyPeerAvailable()
	}
}

func scorePeer(p peer.Peer) int {
	status := p.Status()
	score := status.PendingRequestCount
	switch status.ConnectionStatus {
	case peer.Available:
		// No penalty
	// case peer.ConnectRetrying:
	// 	score += _retryingPenalty
	case peer.Connecting:
		score += _connectingPenalty
	default:
		score += _otherUnconnectedPenalty
	}
	return score
}
