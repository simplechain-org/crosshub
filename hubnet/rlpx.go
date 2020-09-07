package hubnet

import (
	"errors"
	"github.com/simplechain-org/go-simplechain/log"
	"github.com/simplechain-org/go-simplechain/rlp"
	"io"
)

const (
	maxUint24 = ^uint32(0) >> 8
)

type p2pRW struct {
	conn io.ReadWriter
}

func newp2pRW(conn io.ReadWriter) *p2pRW {
	return &p2pRW{
		conn:       conn,
	}
}

func (rw *p2pRW) WriteMsg(msg *Msg) error {
	ptype, err := rlp.EncodeToBytes(msg.Code)
	if err != nil {
		return err
	}
	headbuf := make([]byte, 32)
	fsize := uint32(len(ptype)) + msg.Size
	if fsize > maxUint24 {
		return errors.New("message size overflows uint24")
	}
	putInt24(fsize, headbuf) // TODO: check overflow
	if _, err := rw.conn.Write(headbuf); err != nil {
		log.Info("headbuf","err",err)
		return err
	}

	if _, err := rw.conn.Write(ptype); err != nil {
		log.Info("ptype","err",err)
		return err
	}

	//payload, err := ioutil.ReadAll(msg.Payload)
	//if err != nil {
	//	log.Info("ReadAll","er",err)
	//	return err
	//}
	if _, err := rw.conn.Write(msg.Bytes); err != nil {
		log.Info("payload","err",err)
		return err
	}
	return nil
}

func (rw *p2pRW) ReadMsg() (msg Msg, err error) {
	// read the header
	headbuf := make([]byte, 32)
	if _, err := io.ReadFull(rw.conn, headbuf); err != nil {
		log.Info("ReadMsg1","err",err)
		return msg, err
	}
	fsize := readInt24(headbuf)
	codebuf := make([]byte, 1)
	if _, err := io.ReadFull(rw.conn, codebuf); err != nil {
		log.Info("ReadMsg2","err",err)
		return msg, err
	}
	err = rlp.DecodeBytes(codebuf,&msg.Code)
	if err != nil {
		log.Info("DecodeBytes","err",err)
		return msg,err
	}
	msg.Size = fsize - 1
	framebuf := make([]byte, msg.Size)
	if _, err := io.ReadFull(rw.conn, framebuf); err != nil {
		log.Info("ReadMsg2","err",err)
		return msg, err
	}
	msg.Bytes = framebuf


	// decode message code
	//content := bytes.NewReader(framebuf)
	////msg.Code长度固定
	//if err := rlp.Decode(content, &msg.Code); err != nil {
	//	log.Info("Decode","err",err)
	//	return msg, err
	//}
	//content读出去后长度变化
	//msg.Size = uint32(content.Len())
	//msg.Payload = content
	//msg.Bytes,err  = ioutil.ReadAll(content)
	//if err != nil {
	//	log.Info("ReadAll","err",err)
	//}
	return msg, nil
}

func readInt24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

func putInt24(v uint32, b []byte) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}