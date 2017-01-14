package peerheap

import "go.uber.org/yarpc/api/peer"

type peerScore struct {
	list        *List
	peer        peer.Peer
	id          peer.Identifier
	status      peer.Status
	score       int
	idx         int // index in the peer list.
	last        int // snapshot of the heap's incrementing counter.
	boundFinish func(error)
}

func (ps *peerScore) NotifyStatusChanged(_ peer.Identifier) {
	status := ps.peer.Status()
	if ps.status == status {
		return
	}
	ps.status = status
	ps.list.notifyStatusChanged(ps)
}

func (ps *peerScore) finish(error) {
	// TODO update pending request count and inform peer list to rescore.
	// currently we rely on the transport to send a notification of pending
	// count change, but we could anticipate it here.
}

type _noSub struct{}

func (_noSub) NotifyStatusChanged(peer.Identifier) {}

var noSub = _noSub{}
