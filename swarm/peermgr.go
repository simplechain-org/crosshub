package swarm

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/simplechain-org/crosshub/hubnet"
)

type PeerManager interface {
	// Start
	Start() error

	// Stop
	Stop() error

	// AsyncSend sends message to peer with peer info.
	AsyncSend(uint64, *hubnet.Msg) error

	// SendWithStream sends message using existed stream
	SendWithStream(network.Stream, *hubnet.Msg) error

	// Send sends message waiting response
	Send(uint64, *hubnet.Msg) (*hubnet.Msg, error)

	// Broadcast message to all node
	Broadcast(*hubnet.Msg) error

	// Peers
	Peers() map[uint64]*peer.AddrInfo

	// OtherPeers
	OtherPeers() map[uint64]*peer.AddrInfo

	// SubscribeOrderMessage
	//SubscribeOrderMessage(ch chan<- events.OrderMessageEvent) event.Subscription
}
