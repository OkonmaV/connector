package connector

import (
	"context"
	"net"
	"sync"
	"time"
)

type EpollReConnector[Tm any, PTm interface {
	Readable
	*Tm
}] struct {
	connector  *EpollConnector[Tm, PTm]
	msghandler MessageHandler[PTm]

	mux       sync.Mutex
	isstopped bool

	reconAddr Addr

	doOnDial         func(net.Conn) error // right after dial, before NewEpollConnector() call
	doAfterReconnect func() error         // after StartServing() call
}

type Addr struct {
	netw    string
	address string
}

type connectable interface {
	connect() (succeeded bool)
}

func NewEpollReConnector[Tmessage any, PTmessage interface {
	Readable
	*Tmessage
}, Thandler MessageHandler[PTmessage]](conn net.Conn, messagehandler Thandler, doOnDial func(net.Conn) error, doAfterReconnect func() error) (*EpollReConnector[Tmessage, PTmessage], error) {
	reconn := &EpollReConnector[Tmessage, PTmessage]{
		msghandler: messagehandler,
		reconAddr:  Addr{netw: conn.RemoteAddr().Network(), address: conn.RemoteAddr().String()},

		doOnDial:         doOnDial,
		doAfterReconnect: doAfterReconnect,
	}

	var err error
	if reconn.connector, err = NewEpollConnector[Tmessage, PTmessage](conn, reconn); err != nil {
		return nil, err
	}

	return reconn, nil
}

// NO NETW & ADDR VALIDATION
func AddToReconnectionQueue[Tmessage any, PTmessage interface {
	Readable
	*Tmessage
}, Thandler MessageHandler[PTmessage]](netw string, addr string, messagehandler Thandler, doOnDial func(net.Conn) error, doAfterReconnect func() error) *EpollReConnector[Tmessage, PTmessage] {
	reconn := &EpollReConnector[Tmessage, PTmessage]{
		connector:  &EpollConnector[Tmessage, PTmessage]{isclosed: true},
		reconAddr:  Addr{netw: netw, address: addr},
		msghandler: messagehandler,

		doOnDial:         doOnDial,
		doAfterReconnect: doAfterReconnect,
	}
	recon_req_ch <- reconn
	return reconn
}

var recon_req_ch chan connectable
var dialer *net.Dialer
var Reconnection_dial_timeout = time.Millisecond * 500

func SetupReconnection(ctx context.Context, recon_check_ticktime time.Duration, reconbuf_targetsize, queuechan_size int) {
	if reconbuf_targetsize == 0 || queuechan_size == 0 {
		panic("target buffer size && queue size must be > 0")
	}
	if recon_check_ticktime == 0 {
		panic("zero recon_check_ticktime")
	}
	if recon_req_ch != nil {
		panic("reconnection already set up")
	}
	recon_req_ch = make(chan connectable, queuechan_size)

	go serveReconnects(ctx, recon_check_ticktime, reconbuf_targetsize)
}

// можно в канал еще пихать протокол-адрес для реконнекта, если будут возможны случаи переезда сервиса, или не мы инициировали подключение
func serveReconnects(ctx context.Context, ticktime time.Duration, targetbufsize int) { // TODO: test this
	buf := make([]connectable, 0, targetbufsize)
	ticker := time.NewTicker(ticktime)
	dialer = &net.Dialer{Timeout: Reconnection_dial_timeout}

	for {
		select {
		case <-ctx.Done():
			return
		case req := <-recon_req_ch:
			buf = append(buf, req)
		case <-ticker.C:
			for i := 0; i < len(buf); i++ {
				if buf[i].connect() {
					buf = buf[:i+copy(buf[i:], buf[i+1:])] // трем из буфера
					i--
				}
			}
			if cap(buf) > targetbufsize && len(buf) <= targetbufsize { // при переполнении буфера снова его уменьшаем, если к этому моменту разберемся с реконнектами // защиту от переполнения буфера ставить нельзя, иначе куда оверфловнутые реконнекты пихать
				newbuf := make([]connectable, len(buf), targetbufsize)
				copy(newbuf, buf)
				buf = newbuf
			}
		}
	}
}

func (recon *EpollReConnector[Tm, PTm]) connect() (succeeded bool) {
	recon.mux.Lock()
	if !recon.isstopped {
		if recon.connector.IsClosed() {
			conn, err := dialer.Dial(recon.reconAddr.netw, recon.reconAddr.address)
			if err != nil {
				recon.mux.Unlock()
				return // не логается
			}
			if recon.doOnDial != nil {
				if err := recon.doOnDial(conn); err != nil {
					conn.Close()
					recon.mux.Unlock()
					return
				}
			}
			conn.SetReadDeadline(time.Time{}) //TODO: нужно ли обнулять conn.readtimeout после doOnDial() ??

			newcon, err := NewEpollConnector[Tm, PTm](conn, recon)
			if err != nil {
				conn.Close()
				recon.mux.Unlock()
				return // не логается
			}

			if err = newcon.StartServing(); err != nil {
				newcon.ClearFromCache()
				conn.Close()
				recon.mux.Unlock()
				return // не логается
			}

			recon.connector = newcon

			if recon.doAfterReconnect != nil {
				if err := recon.doAfterReconnect(); err != nil {
					recon.connector.Close(err)
					recon.mux.Unlock()
					return
				}
			}
		}
	}
	recon.mux.Unlock()
	return true
}

func (reconnector *EpollReConnector[_, _]) StartServing() error {
	return reconnector.connector.StartServing()
}

func (connector *EpollReConnector[_, _]) ClearFromCache() {
	connector.mux.Lock()
	defer connector.mux.Unlock()

	connector.connector.ClearFromCache()
}

func (a Addr) Network() string {
	return a.netw
}
func (a Addr) String() string {
	return a.address
}

func (reconnector *EpollReConnector[_, PTm]) Handle(message PTm) error {
	return reconnector.msghandler.Handle(message)
}

func (reconnector *EpollReConnector[_, _]) HandleClose(reason error) {

	reconnector.msghandler.HandleClose(reason)

	if !reconnector.isstopped { // нет мьютекса, т.к. невозможна конкуренция
		recon_req_ch <- reconnector
	}
}
func (reconnector *EpollReConnector[_, _]) Send(message []byte) error {
	return reconnector.connector.Send(message)
}

// doesn't stop reconnection
func (reconnector *EpollReConnector[_, _]) Close(reason error) {
	reconnector.mux.Lock()
	defer reconnector.mux.Unlock()

	reconnector.connector.Close(reason)

}

// call in HandleClose() will cause deadlock
func (reconnector *EpollReConnector[_, _]) IsClosed() bool {
	return reconnector.connector.IsClosed()
}

func (reconnector *EpollReConnector[_, _]) RemoteAddr() net.Addr {
	return reconnector.reconAddr
}

func (reconnector *EpollReConnector[_, _]) IsReconnectStopped() bool { // только извне! иначе потенциальная блокировка
	reconnector.mux.Lock()
	defer reconnector.mux.Unlock()
	return reconnector.isstopped
}

// DOES NOT CLOSE CONN
func (reconnector *EpollReConnector[_, _]) CancelReconnect() { // только извне! иначе потенциальная блокировка
	reconnector.mux.Lock()
	defer reconnector.mux.Unlock()
	reconnector.isstopped = true
}
