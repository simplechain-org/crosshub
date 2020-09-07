package hubnet

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

const protocolID protocol.ID = "/simplechain/test/1.0"

func TestP2P_Connect(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6001)
	p2, addr2 := generateNetwork(t, 6002)

	err := p1.Connect(addr2)
	if err != nil {
		t.Error(err)
	}
	err = p2.Connect(addr1)
	if err != nil {
		t.Error(err)
	}
}

func TestP2p_ConnectWithNullIDStore(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6003)
	p2, addr2 := generateNetwork(t, 6004)

	err := p1.Connect(addr2)
	if err != nil {
		t.Error(err)
	}
	err = p2.Connect(addr1)
	if err != nil {
		t.Error(err)
	}
}

func TestP2P_Send(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6005)
	p2, addr2 := generateNetwork(t, 6006)
	var err error
	var msg *Msg
	msg,err = NewMsg(1,[]byte("Good Afternoon!"))
	if err != nil {
		t.Error(err)
	}

	ch := make(chan struct{})

	p2.SetMessageHandler(func(s network.Stream, g *Msg) {
		var word []byte
		g.Decode(&word)
		t.Log(msg.Size == g.Size,g.Code,string(word))
		close(ch)
	})

	err = p1.Start()
	if err != nil {
		t.Error(err)
	}
	err = p2.Start()
	if err != nil {
		t.Error(err)
	}

	err = p1.Connect(addr2)
	if err != nil {
		t.Error(err)
	}
	err = p2.Connect(addr1)
	if err != nil {
		t.Error(err)
	}

	err = p1.AsyncSend(addr2, msg)
	if err != nil {
		t.Error(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ch:
		return
	case <-ctx.Done():
		t.Error(fmt.Errorf("timeout"))
		return
	}
}

func TestP2p_MultiSend(t *testing.T) {
	p1, addr1 := generateNetwork(t, 6007)
	p2, addr2 := generateNetwork(t, 6008)
	var err error
	err = p1.Start()
	if err != nil {
		t.Error(err)
	}
	err = p2.Start()
	if err != nil {
		t.Error(err)
	}

	err = p1.Connect(addr2)
	if err != nil {
		t.Error(err)
	}
	err = p2.Connect(addr1)
	if err != nil {
		t.Error(err)
	}

	pubKey, err := p2.GetRemotePubKey(addr1.ID)
	if err != nil {
		t.Error(err)
	}

	id, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(addr1.ID ,id)
	}

	//raw, err := pubKey.Raw()
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//key, err := x509.ParsePKIXPublicKey(raw)
	//if err != nil {
	//	t.Error(err)
	//}
	//
	//publicKey, err := ecdsa1.NewPublicKey(*key.(*ecdsa2.PublicKey))
	//if err != nil {
	//	t.Error(err)
	//}
	//add , err := publicKey.Address()
	//if err != nil {
	//	t.Error(err)
	//} else {
	//	t.Log(add)
	//}

	N := 50
	var msg *Msg
	msg,err = NewMsg(1,[]byte("Good Afternoon!"))
	if err != nil {
		t.Error(err)
	}
	count := 0
	ch := make(chan struct{})

	p2.SetMessageHandler(func(s network.Stream, g *Msg) {
		var word []byte
		g.Decode(&word)
		count++
		t.Log(msg.Size == g.Size,g.Code,string(word),count)
		if count == N {
			close(ch)
			return
		}
	})

	go func() {
		for i := 0; i < N; i++ {
			time.Sleep(200 * time.Microsecond)
			err = p1.AsyncSend(addr2, msg)
			if err != nil {
				t.Error(err)
			}
		}

	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ch:
		return
	case <-ctx.Done():
		t.Error(fmt.Errorf("timeout"))
	}
}

func generateNetwork(t *testing.T, port int) (Network, *peer.AddrInfo) {
	privKey, pubKey, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		t.Error(err)
	}

	pid1, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		t.Error(err)
	}
	addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)
	maddr := fmt.Sprintf("%s/p2p/%s", addr, pid1)
	p2p, err := New(
		WithLocalAddr(addr),
		WithPrivateKey(privKey),
		WithProtocolID(protocolID),
	)
	if err != nil {
		t.Error(err)
	}

	info, err := AddrToPeerInfo(maddr)
	if err != nil {
		t.Error(err)
	}

	return p2p, info
}