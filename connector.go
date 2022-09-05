package connector

import (
	"errors"
	"net"
	"time"
)

var ErrWeirdData error = errors.New("weird data")
var ErrEmptyPayload error = errors.New("empty payload")
var ErrClosedConnector error = errors.New("closed connector")
var ErrNilConn error = errors.New("conn is nil")
var ErrNilGopool error = errors.New("gopool is nil, setup gopool first")
var ErrReadTimeout error = errors.New("read timeout")

// for user's implementation
// lib project/dynamicworkerspool
type PoolScheduler interface {
	Schedule(task func())
	ScheduleWithTimeout(task func(), timeout time.Duration) error
}

// for user's implementation
type Readable interface {
	Read(conn net.Conn) error
}

// for user's implementation
type MessageHandler[T Readable] interface {
	Handle(T) error
	HandleClose(reason error)
}

// implemented by connector
type Conn interface {
	StartServing() error
	ClearFromCache()
	Informer
	Closer
	Sender
}

// implemented by connector
type Sender interface {
	Send(rawmsg []byte) error
}

// implemented by connector
type Informer interface {
	RemoteAddr() net.Addr
	IsClosed() bool
}

// implemented by connector
type Closer interface {
	Close(error)
}

// implemented by connector
type ReConn interface {
	Conn
	//ReconnectedItself(conn net.Conn) error
	IsReconnectStopped() bool
	CancelReconnect()
}
