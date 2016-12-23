package circus

import (
	"context"
	"testing"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/x/roundrobin"
	"go.uber.org/yarpc/transport/http"

	"github.com/stretchr/testify/assert"
)

func BenchmarkCircus(b *testing.B) {
	x := http.NewTransport()
	pl := New(x)
	pl.Update(peer.ListUpdates{
		Additions: []peer.Identifier{hostport.PeerIdentifier("127.0.0.1:80")},
	})
	pl.Start()
	benchPeerChooser(b, pl)
}

func BenchmarkRoundRobin(b *testing.B) {
	x := http.NewTransport()
	pl := roundrobin.New(x)
	pl.Update(peer.ListUpdates{
		Additions: []peer.Identifier{hostport.PeerIdentifier("127.0.0.1:80")},
	})
	pl.Start()
	defer pl.Stop()
	benchPeerChooser(b, pl)
}

func benchPeerChooser(b *testing.B, pc peer.Chooser) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, finish, err := pc.Choose(ctx, nil)
		if assert.NoError(b, err, "should not be an error") {
			// TODO defer finish randomly
			finish(nil)
		}
	}
}
