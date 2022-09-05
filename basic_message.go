package connector

import (
	"encoding/binary"
	"errors"
	"net"
	"time"
)

const maxlength = 4096

type BasicMessage struct {
	Payload []byte
}

var errReadedLess error = errors.New("readed less bytes than expected")

func (msg *BasicMessage) Read(conn net.Conn) error {

	buf := make([]byte, 4)
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	n, err := conn.Read(buf)
	if err != nil {
		return err
	}
	if n != 4 {
		return errReadedLess
	}
	msglength := binary.LittleEndian.Uint32(buf)
	if msglength > maxlength {
		return errors.New("payload too long")
	}
	msg.Payload = make([]byte, msglength)
	//conn.SetReadDeadline(time.Now().Add((time.Millisecond * 700) * (time.Duration((msglength / 1024) + 1))))
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	if n, err = conn.Read(msg.Payload); err != nil {
		return err
	}
	//fmt.Println("CONNECTOR MESSAGE READED: ", msg.Payload, " LEN: ", msglength) //////////////////////////////
	if uint32(n) != msglength {
		return errReadedLess
	}
	return nil
}

// payload not allocated
func NewBasicMessage() *BasicMessage {
	return &BasicMessage{}
}

func FormatBasicMessage(message []byte) []byte {
	formattedmsg := make([]byte, 4+len(message))
	// if len(message) > 0 {
	binary.LittleEndian.PutUint32(formattedmsg, uint32(len(message)))
	copy(formattedmsg[4:], message)
	return formattedmsg
	// 	}
	// 	return formattedmsg
}
