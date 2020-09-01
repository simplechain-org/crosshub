package swarm

import (
	"crypto/x509"
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/simplechain-org/crosshub/hubnet"
	"github.com/simplechain-org/crosshub/cert"
	"github.com/simplechain-org/go-simplechain/log"
	"github.com/simplechain-org/go-simplechain/rlp"
)

func (swarm *Swarm) handleMessage(s network.Stream, data *hubnet.Msg) {
	handler := func() error {
		switch data.Code {
		case 1:
			var word []byte
			data.Decode(&word)
			log.Info("handler","msg",string(word))
			var msg hubnet.Msg
			if size, r, err := rlp.EncodeToReader([]byte( fmt.Sprintf("Yes!,I am %s",swarm.repo.Key.Address)));err != nil {
				log.Error("EncodeToReader","err",err)
			} else {
				msg = hubnet.Msg{Code: 2, Size: uint32(size), Payload: r}
			}
			swarm.SendWithStream(s,&msg)
		//case pb.Message_GET_BLOCK:
		//	return swarm.handleGetBlockPack(s, m)
		//case pb.Message_FETCH_CERT:
		//	return swarm.handleFetchCertMessage(s)
		//case pb.Message_CONSENSUS:
		//	go swarm.orderMessageFeed.Send(events.OrderMessageEvent{Data: m.Data})
		//case pb.Message_FETCH_BLOCK_SIGN:
		//	swarm.handleFetchBlockSignMessage(s, m.Data)
		default:
			//swarm.logger.WithField("module", "p2p").Errorf("can't handle msg[type: %v]", m.Type)
			//log
			return nil
		}

		return nil
	}

	if err := handler(); err != nil {
		log.Info("handler","err",err)
	}
}

//func (swarm *Swarm) handleFetchCertMessage(s network.Stream) error {
//	certs := &model.CertsMessage{
//		AgencyCert: swarm.repo.Certs.AgencyCertData,
//		NodeCert:   swarm.repo.Certs.NodeCertData,
//	}
//
//	data, err := certs.Marshal()
//	if err != nil {
//		return fmt.Errorf("marshal certs: %w", err)
//	}
//
//	msg := &pb.Message{
//		Type: pb.Message_FETCH_CERT,
//		Data: data,
//	}
//
//	err = swarm.SendWithStream(s, msg)
//	if err != nil {
//		return fmt.Errorf("send msg: %w", err)
//	}
//
//	return nil
//}

func verifyCerts(nodeCert *x509.Certificate, agencyCert *x509.Certificate, caCert *x509.Certificate) error {
	if err := cert.VerifySign(agencyCert, caCert); err != nil {
		return fmt.Errorf("verify agency cert: %w", err)
	}

	if err := cert.VerifySign(nodeCert, agencyCert); err != nil {
		return fmt.Errorf("verify node cert: %w", err)
	}

	return nil
}