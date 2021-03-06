// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package peerheap

import (
	"context"
	"math"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	yerrors "go.uber.org/yarpc/internal/errors"
	ysync "go.uber.org/yarpc/internal/sync"
)

const unavailablePenalty = math.MaxInt32

// List is a peer list and peer chooser that favors the peer with the least
// pending requests, and then favors the least recently used or most recently
// introduced peer.
type List struct {
	mu   sync.Mutex
	once ysync.LifecycleOnce

	transport peer.Transport

	byScore      peerHeap
	byIdentifier map[string]*peerScore

	peerAvailableEvent chan struct{}
}

// IsRunning returns whether the peer list is running.
func (pl *List) IsRunning() bool {
	return pl.once.IsRunning()
}

// Start starts the peer list.
func (pl *List) Start() error {
	return pl.once.Start(nil)
}

// Stop stops the peer list. This releases all retained peers.
func (pl *List) Stop() error {
	return pl.once.Stop(pl.clearPeers) // TODO clear peers
}

// New returns a new peer heap-chooser-list for the given transport.
func New(transport peer.Transport) *List {
	return &List{
		once:               ysync.Once(),
		transport:          transport,
		byIdentifier:       make(map[string]*peerScore),
		peerAvailableEvent: make(chan struct{}, 1),
	}
}

// Update satisfies the peer.List interface, so a peer provider can manage the
// retained peers.
func (pl *List) Update(updates peer.ListUpdates) error {
	var errs []error

	pl.mu.Lock()
	defer pl.mu.Unlock()

	for _, pid := range updates.Removals {
		if err := pl.releasePeer(pid); err != nil {
			errs = append(errs, err)
		}
	}
	for _, pid := range updates.Additions {
		if err := pl.retainPeer(pid); err != nil {
			errs = append(errs, err)
		}
	}

	return yerrors.MultiError(errs)
}

// retainPeer must be called with the mutex locked.
func (pl *List) retainPeer(pid peer.Identifier) error {
	if _, ok := pl.byIdentifier[pid.Identifier()]; ok {
		return peer.ErrPeerAddAlreadyInList(pid.Identifier())
	}

	ps := &peerScore{id: pid, list: pl}
	p, err := pl.transport.RetainPeer(pid, ps)
	if err != nil {
		return err
	}

	ps.peer = p
	ps.score = scorePeer(p)
	ps.boundFinish = ps.finish
	pl.byIdentifier[pid.Identifier()] = ps
	pl.byScore.pushPeer(ps)
	pl.internalNotifyStatusChanged(ps)
	return nil
}

// releasePeer must be called with the mutex locked.
func (pl *List) releasePeer(pid peer.Identifier) error {
	ps, ok := pl.byIdentifier[pid.Identifier()]
	if !ok {
		return peer.ErrPeerRemoveNotInList(pid.Identifier())
	}

	if err := pl.byScore.validate(ps); err != nil {
		return err
	}

	err := pl.transport.ReleasePeer(pid, ps)
	delete(pl.byIdentifier, pid.Identifier())
	pl.byScore.delete(ps.idx)
	ps.list = nil
	return err
}

func (pl *List) clearPeers() error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	var errs []error

	for {
		ps, ok := pl.byScore.peekPeer()
		if !ok {
			break
		}
		if err := pl.releasePeer(ps.id); err != nil {
			errs = append(errs, err)
		}
	}

	return yerrors.MultiError(errs)
}

// Choose satisfies peer.Chooser, providing a single peer for a request, a
// callback for when the request is finished, or an error if it fails.
// The choose method takes a context that must have a deadline.
// Choose resepects this deadline, waiting for an available peer until the
// deadline.
// The peer heap does not use the given *transport.Request and can safely
// receive nil.
func (pl *List) Choose(ctx context.Context, _ *transport.Request) (peer.Peer, func(error), error) {
	if err := pl.once.WhenRunning(ctx); err != nil {
		return nil, nil, err
	}

	for {
		if ps, ok := pl.get(); ok {
			pl.notifyPeerAvailable()
			ps.peer.StartRequest()
			return ps.peer, ps.boundFinish, nil
		}

		if err := pl.waitForPeerAvailableEvent(ctx); err != nil {
			return nil, nil, err
		}
	}
}

func (pl *List) get() (*peerScore, bool) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	ps, ok := pl.byScore.popPeer()
	if !ok {
		return nil, false
	}

	// Note: We push the peer back to reset the "next" counter.
	// This gives us round-robin behavior.
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

// NotifyStatusChanged receives notifications when a peer becomes available,
// connected, unavailable, or when its pending request count changes.
// This method satisfies peer.Subscriber and is only used for tests, since
// the peer heap has a subscriber for each invividual peer.
func (pl *List) NotifyStatusChanged(pid peer.Identifier) {
	pl.mu.Lock()
	ps := pl.byIdentifier[pid.Identifier()]
	pl.mu.Unlock()
	ps.NotifyStatusChanged(pid)
}

func (pl *List) peerScoreChanged(ps *peerScore) {
	pl.mu.Lock()
	pl.rescorePeer(ps)
	pl.mu.Unlock()

	if ps.peer.Status().ConnectionStatus == peer.Available {
		pl.notifyPeerAvailable()
	}
}

func (pl *List) internalNotifyStatusChanged(ps *peerScore) {
	pl.rescorePeer(ps)

	if ps.peer.Status().ConnectionStatus == peer.Available {
		pl.notifyPeerAvailable()
	}
}

func (pl *List) rescorePeer(ps *peerScore) {
	p := ps.peer
	ps.status = p.Status()
	ps.score = scorePeer(p)
	pl.byScore.update(ps.idx)
}

func scorePeer(p peer.Peer) int64 {
	status := p.Status()
	score := int64(status.PendingRequestCount)
	if status.ConnectionStatus != peer.Available {
		score += int64(unavailablePenalty)
	}
	return score
}
