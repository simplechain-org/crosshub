package hubnet

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/simplechain-org/go-simplechain/rlp"
)

const (
	errInvalidMsgCode = iota
	errInvalidMsg
	errNotInRaftCluster = iota + 100
)

var errorToString = map[int]string{
	errInvalidMsgCode: "invalid message code",
	errInvalidMsg:     "invalid message",
	// Quorum
	errNotInRaftCluster: "not in raft cluster",
}

// Msg defines the structure of a p2p message.
//
// Note that a Msg can only be sent once since the Payload reader is
// consumed during sending. It is not possible to create a Msg and
// send it any number of times. If you want to reuse an encoded
// structure, encode the payload into a byte array and create a
// separate Msg with a bytes.Reader as Payload for each send.
type Msg struct {
	Code       uint64
	Size       uint32 // Size of the raw payload
	Payload    io.Reader
	ReceivedAt time.Time
}

func NewMsg(msgcode uint64, data interface{}) (*Msg,error) {
	size, r, err := rlp.EncodeToReader(data)
	if err != nil {
		return nil,err
	}
	return  &Msg{Code: msgcode, Size: uint32(size), Payload: r} , nil
}

// Decode parses the RLP content of a message into
// the given value, which must be a pointer.
//
// For the decoding rules, please see package rlp.
func (msg Msg) Decode(val interface{}) error {
	s := rlp.NewStream(msg.Payload, uint64(msg.Size))
	if err := s.Decode(val); err != nil {
		return err
		//return newPeerError(errInvalidMsg, "(code %x) (size %d) %v", msg.Code, msg.Size, err)
	}
	return nil
}

func (msg Msg) String() string {
	return fmt.Sprintf("msg #%v (%v bytes)", msg.Code, msg.Size)
}

// Discard reads any remaining payload data into a black hole.
func (msg Msg) Discard() error {
	_, err := io.Copy(ioutil.Discard, msg.Payload)
	return err
}

//type MsgReader interface {
//	ReadMsg() (Msg, error)
//}
//
//type MsgWriter interface {
//	// WriteMsg sends a message. It will block until the message's
//	// Payload has been consumed by the other end.
//	//
//	// Note that messages can be sent only once because their
//	// payload reader is drained.
//	WriteMsg(Msg) error
//}
//
//// MsgReadWriter provides reading and writing of encoded messages.
//// Implementations should ensure that ReadMsg and WriteMsg can be
//// called simultaneously from multiple goroutines.
//type MsgReadWriter interface {
//	MsgReader
//	MsgWriter
//}