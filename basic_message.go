package connector

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"
)

const maxlength = 2048

type BasicMessage struct {
	Payload []byte
}

func (msg *BasicMessage) Read(conn net.Conn) error {

	buf := make([]byte, 4)
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return err
	}
	msglength := binary.LittleEndian.Uint32(buf)
	if msglength > maxlength {
		return errors.New("payload too long")
	}
	msg.Payload = make([]byte, msglength)
	//conn.SetReadDeadline(time.Now().Add((time.Millisecond * 700) * (time.Duration((msglength / 1024) + 1))))
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	if _, err = io.ReadFull(conn, msg.Payload); err != nil {
		return err
	}
	return nil
}

func (msg *BasicMessage) ReadWithoutDeadline(conn net.Conn) error {

	buf := make([]byte, 4)
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return err
	}
	msglength := binary.LittleEndian.Uint32(buf)
	if msglength > maxlength {
		return errors.New("payload too long")
	}
	msg.Payload = make([]byte, msglength)
	if _, err = io.ReadFull(conn, msg.Payload); err != nil {
		return err
	}
	return nil
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
