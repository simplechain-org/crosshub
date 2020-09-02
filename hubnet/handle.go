package hubnet

import (
	"context"
	"fmt"
	"github.com/simplechain-org/go-simplechain/log"
	"io"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
)

// handle newly connected stream
func (p2p *P2P) handleNewStream(s network.Stream) {
	if err := s.SetReadDeadline(time.Time{}); err != nil {
		//p2p.logger.WithField("error", err).Error("Set stream read deadline")
		return
	}
	rw := newp2pRW(s)
	for {
		var msg Msg
		var err error
		if msg,err = rw.ReadMsg(); err != nil {
			if err != io.EOF {
				if err := s.Reset(); err != nil {
					log.Error("Reset stream",err)
				}
			}
			log.Error("readMsg", "err",err)
			return
		}

		if p2p.handleMessage != nil {
			p2p.handleMessage(s, &msg)
		}
	}
}

// waitMsg wait the incoming messages within time duration.
func waitMsg(stream network.Stream, timeout time.Duration) *Msg {
	rw := newp2pRW(stream)
	ch := make(chan *Msg)
	go func() {
		if msg,err := rw.ReadMsg(); err == nil {
			ch <- &msg
		} else {
			ch <- nil
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	select {
	case r := <-ch:
		cancel()
		return r
	case <-ctx.Done():
		cancel()
		return nil
	}
}

func (p2p *P2P) send(s network.Stream, msg *Msg) error {
	deadline := time.Now().Add(sendTimeout)

	if err := s.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	rw := newp2pRW(s)

	if err := rw.WriteMsg(msg); err != nil {
		return fmt.Errorf("write msg: %w", err)
	}

	return nil
}