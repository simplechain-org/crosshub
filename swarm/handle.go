package swarm

import (
	"crypto/x509"
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/simplechain-org/crosshub/cert"
	"github.com/simplechain-org/crosshub/core"
	"github.com/simplechain-org/crosshub/hubnet"
	"github.com/simplechain-org/go-simplechain/log"
)

func (swarm *Swarm) handleMessage(s network.Stream, data *hubnet.Msg) {
	handler := func() error {
		switch data.Code {
		case GetCertMsg:
			var certs CertsMessage
			if err := data.Decode(&certs); err != nil {
				log.Info("Decode msg","err",err)
				return err
			}


			nodeCert, err := cert.ParseCert(certs.NodeCert)
			if err != nil {
				log.Info("ParseCert","err",err)
				return fmt.Errorf("parse node cert: %w", err)
			}

			agencyCert, err := cert.ParseCert(certs.AgencyCert)
			if err != nil {
				log.Info("ParseCert","err",err)
				return fmt.Errorf("parse agency cert: %w", err)
			}
			if err := verifyCerts(nodeCert, agencyCert, swarm.repo.Certs.CACert); err != nil {
				log.Info("ParseCert","err",err)
				return fmt.Errorf("verify certs: %w", err)
			}

			for _, addr := range swarm.peers {
				if addr.ID.String() ==  certs.Id {
					swarm.connectedPeers.Store(addr.ID,addr)
				}
			}
			//swarm.connectedPeers.Store()
			//TODO 网络拓展 swarm.connectedPeers.RemoteStore(certs.Id,addr)
			return swarm.handleFetchCertMessage(s)
		case CertMsg:


		case CtxSignMsg:
			var ev core.CrossTransaction
			data.Decode(&ev)
			swarm.messageCh <- &ev
		case RtxSignMsg:
			var er core.ReceptTransaction
			data.Decode(&er)
			swarm.messageCh <- &er
		default:
			log.Info("can't handle msg","code",data.Code)
			return nil
		}

		return nil
	}

	if err := handler(); err != nil {
		log.Info("handler","err",err)
	}
}

type CertsMessage struct {
	Id         string
	AgencyCert []byte
	NodeCert   []byte
}

func (swarm *Swarm) handleFetchCertMessage(s network.Stream) error {
	certs := &CertsMessage{
		Id:         swarm.repo.NetworkConfig.PeerId,
		AgencyCert: swarm.repo.Certs.AgencyCertData,
		NodeCert:   swarm.repo.Certs.NodeCertData,
	}
	msg, err := hubnet.NewMsg(CertMsg,certs)
	if err != nil {
		return err
	}
	err = swarm.SendWithStream(s, msg)
	if err != nil {
		return fmt.Errorf("send msg: %w", err)
	}
	return nil
}

func verifyCerts(nodeCert *x509.Certificate, agencyCert *x509.Certificate, caCert *x509.Certificate) error {
	if err := cert.VerifySign(agencyCert, caCert); err != nil {
		return fmt.Errorf("verify agency cert: %w", err)
	}

	if err := cert.VerifySign(nodeCert, agencyCert); err != nil {
		return fmt.Errorf("verify node cert: %w", err)
	}

	return nil
}