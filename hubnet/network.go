package hubnet

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type ConnectCallback func(*peer.AddrInfo) error


type MessageHandler func(network.Stream, *Msg)

type Network interface {
	// Start start the network service.
	Start() error

	// Stop stop the network service.
	Stop() error

	// Connect connects peer by ID.
	Connect(*peer.AddrInfo) error

	// Disconnect peer with id
	Disconnect(*peer.AddrInfo) error

	// SetConnectionCallback sets the callback after connecting
	SetConnectCallback(ConnectCallback)

	// SetMessageHandler sets message handler
	SetMessageHandler(MessageHandler)

	// AsyncSend sends message to peer with peer info.
	AsyncSend(*peer.AddrInfo, *Msg) error

	// Send message using existed stream
	SendWithStream(network.Stream, *Msg) error

	// Send sends message waiting response
	Send(*peer.AddrInfo, *Msg) (*Msg, error)

	// Broadcast message to all node
	Broadcast([]*peer.AddrInfo, *Msg) error

	// GetRemotePubKey gets remote public key
	GetRemotePubKey(id peer.ID) (crypto.PubKey, error)
}
