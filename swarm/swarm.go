package swarm

import (
	"context"
	"fmt"
	"github.com/simplechain-org/crosshub/cert"
	"github.com/simplechain-org/crosshub/core"

	//"github.com/meshplus/bitxhub-kit/network"
	"github.com/simplechain-org/crosshub/hubnet"
	"github.com/simplechain-org/go-simplechain/log"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/simplechain-org/crosshub/repo"
	//"github.com/simplechain-org/crosshub/cert"
)

const (
	//protocolID protocol.ID = "/SimpleChain/CrossHub/1.0.0" // magic protocol
	protocolID protocol.ID = "/SimpleChain/CrossHub/1.0.0"
)

type Swarm struct {
	repo           *repo.Repo
	p2p            hubnet.Network
	peers          map[uint64]*peer.AddrInfo
	connectedPeers sync.Map
	eventCh        chan *core.CrossTransaction

	ctx    context.Context
	cancel context.CancelFunc
}

func New(repo *repo.Repo,ch chan *core.CrossTransaction) (*Swarm, error) {
	p2p, err := hubnet.New(
		hubnet.WithLocalAddr(repo.NetworkConfig.LocalAddr),
		hubnet.WithPrivateKey(repo.Key.Libp2pPrivKey),
		hubnet.WithProtocolID(protocolID),
	)

	if err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Swarm{
		repo:           repo,
		p2p:            p2p,
		peers:          repo.NetworkConfig.OtherNodes,
		connectedPeers: sync.Map{},
		eventCh:        ch,
		ctx:            ctx,
		cancel:         cancel,
	}, nil
}

func (swarm *Swarm) Start() error {
	swarm.p2p.SetMessageHandler(swarm.handleMessage)

	if err := swarm.p2p.Start(); err != nil {
		return err
	}

	log.Info("Start","peers",len(swarm.peers))
	for id, addr := range swarm.peers {
		go func(id uint64, addr *peer.AddrInfo) {
			log.Info("try connet","id",id,"addr",addr.String())
			if err := retry.Retry(func(attempt uint) error {
				if err := swarm.p2p.Connect(addr); err != nil {
					log.Error("p2p.Connect","err",err,"addr",addr.String())
					return err
				}

				if err := swarm.verifyCert(id); err != nil {
					if attempt != 0 && attempt%5 == 0 {
						log.Error("Verify cert","err",err)
					}
					return err
				}

				log.Info("Connect successfully","id",id)

				swarm.connectedPeers.Store(id, addr)

				return nil
			},
				strategy.Wait(1*time.Second),
			); err != nil {
				log.Error("retry.Retry","err",err)
			}
		}(id, addr)
	}
	log.Info("Start successfully")

	go func() {
		for  {
			select {
			case ev := <-swarm.eventCh:
				mm,err := hubnet.NewMsg(3,ev)
				if err != nil {
					log.Info("NewMsg",err)
				}
				swarm.Broadcast(mm)
			//case <-swarm.ctx.Done():
			//	return
			}
		}
	}()

	return nil
}

func (swarm *Swarm) Stop() error {
	swarm.cancel()

	return nil
}

func (swarm *Swarm) AsyncSend(id uint64, msg *hubnet.Msg) error {
	if err := swarm.checkID(id); err != nil {
		return fmt.Errorf("p2p send: %w", err)
	}
	return swarm.p2p.AsyncSend(swarm.peers[id], msg)
}

func (swarm *Swarm) SendWithStream(s network.Stream, msg *hubnet.Msg) error {
	return swarm.p2p.SendWithStream(s, msg)
}

func (swarm *Swarm) Send(id uint64, msg *hubnet.Msg) (*hubnet.Msg, error) {
	if err := swarm.checkID(id); err != nil {
		return nil, fmt.Errorf("check id: %w", err)
	}

	ret, err := swarm.p2p.Send(swarm.peers[id], msg)
	if err != nil {
		return nil, fmt.Errorf("sync send: %w", err)
	}

	return ret, nil
}

func (swarm *Swarm) Broadcast(msg *hubnet.Msg) error {
	var addrs []*peer.AddrInfo
	//for _, addr := range swarm.peers {
	//	log.Info("Broadcast","id",addr.ID.String())
	//	addrs = append(addrs, addr)
	//}
	//log.Info("Broadcast","len",len(addrs))

	swarm.connectedPeers.Range(func(key, value interface{}) bool {
		addr ,ok := value.(*peer.AddrInfo)
		if !ok {
			return false
		}
		addrs = append(addrs,addr)
		return true
	})
	log.Info("Broadcast","len",len(addrs))

	return swarm.p2p.Broadcast(addrs, msg)
}

func (swarm *Swarm) Peers() map[uint64]*peer.AddrInfo {
	m := make(map[uint64]*peer.AddrInfo)
	for id, addr := range swarm.peers {
		m[id] = addr
	}

	return m
}

func (swarm *Swarm) OtherPeers() map[uint64]*peer.AddrInfo {
	m := swarm.Peers()
	delete(m, swarm.repo.NetworkConfig.ID)

	return m
}

//func (swarm *Swarm) SubscribeOrderMessage(ch chan<- events.OrderMessageEvent) event.Subscription {
//	return swarm.orderMessageFeed.Subscribe(ch)
//}

func (swarm *Swarm) verifyCert(id uint64) error {
	if err := swarm.checkID(id); err != nil {
		return fmt.Errorf("check id: %w", err)
	}
	selfCerts := &CertsMessage{
		Id:         swarm.repo.NetworkConfig.PeerId,
		AgencyCert: swarm.repo.Certs.AgencyCertData,
		NodeCert:   swarm.repo.Certs.NodeCertData,
	}

	msg,err := hubnet.NewMsg(GetCertMsg,selfCerts)
	if err != nil {
		return err
	}
	ret,err := swarm.Send(id, msg)
	if err != nil {
		return fmt.Errorf("sync send: %w", err)
	}
	var certs CertsMessage
	ret.Decode(&certs)
	nodeCert, err := cert.ParseCert(certs.NodeCert)
	if err != nil {
		return fmt.Errorf("parse node cert: %w", err)
	}

	agencyCert, err := cert.ParseCert(certs.AgencyCert)
	if err != nil {
		return fmt.Errorf("parse agency cert: %w", err)
	}

	if err := verifyCerts(nodeCert, agencyCert, swarm.repo.Certs.CACert); err != nil {
		return fmt.Errorf("verify certs: %w", err)
	}


	err = swarm.p2p.Disconnect(swarm.peers[id])
	if err != nil {
		return fmt.Errorf("disconnect peer: %w", err)
	}
	return nil
}

func (swarm *Swarm) checkID(id uint64) error {
	if swarm.peers[id] == nil {
		return fmt.Errorf("wrong id: %d", id)
	}

	return nil
}
