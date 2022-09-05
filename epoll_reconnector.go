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
	connector  *EpollConnector[Tm, PTm, *EpollReConnector[Tm, PTm]]
	msghandler MessageHandler[PTm]

	mux       sync.Mutex
	isstopped bool

	reconReqCh ReconnectionProvider[Tm, PTm]
	reconAddr  Addr

	doOnDial         func(net.Conn) error // right after dial, before NewEpollConnector() call
	doAfterReconnect func() error         // after StartServing() call
}

type Addr struct {
	netw    string
	address string
}

type ReconnectionProvider[Tm any, PTm interface {
	Readable
	*Tm
}] chan *EpollReConnector[Tm, PTm]

func NewReconnectionProvider[Tm any, PTm interface {
	Readable
	*Tm
}](ctx context.Context, reconreq_chech_ticktime time.Duration, reconbuf_targetsize int, queuechan_size int) ReconnectionProvider[Tm, PTm] {
	if reconbuf_targetsize == 0 || queuechan_size == 0 {
		panic("target buffer size && queue size must be > 0")
	}
	ch := make(ReconnectionProvider[Tm, PTm], queuechan_size)
	go ch.serveReconnects(ctx, reconreq_chech_ticktime, reconbuf_targetsize)
	return ch
}

func (reconreqch ReconnectionProvider[Tm, PTm]) NewEpollReConnector(conn net.Conn, messagehandler MessageHandler[PTm], doOnDial func(net.Conn) error, doAfterReconnect func() error) (*EpollReConnector[Tm, PTm], error) {
	reconn := &EpollReConnector[Tm, PTm]{
		msghandler: messagehandler,
		reconAddr:  Addr{netw: conn.RemoteAddr().Network(), address: conn.RemoteAddr().String()},
		reconReqCh: reconreqch,

		doOnDial:         doOnDial,
		doAfterReconnect: doAfterReconnect,
	}

	var err error
	if reconn.connector, err = NewEpollConnector[Tm, PTm](conn, reconn); err != nil {
		return nil, err
	}

	return reconn, nil
}

// NO NETW & ADDR VALIDATION
func (reconreqch ReconnectionProvider[Tm, PTm]) AddToReconnectionQueue(netw string, addr string, messagehandler MessageHandler[PTm], doOnDial func(net.Conn) error, doAfterReconnect func() error) *EpollReConnector[Tm, PTm] {
	reconn := &EpollReConnector[Tm, PTm]{
		connector:  &EpollConnector[Tm, PTm, *EpollReConnector[Tm, PTm]]{isclosed: true},
		reconAddr:  Addr{netw: netw, address: addr},
		msghandler: messagehandler,
		reconReqCh: reconreqch,

		doOnDial:         doOnDial,
		doAfterReconnect: doAfterReconnect,
	}
	reconreqch <- reconn
	return reconn
}

var Reconnection_dial_timeout = time.Millisecond * 500

// можно в канал еще пихать протокол-адрес для реконнекта, если будут возможны случаи переезда сервиса, или не мы инициировали подключение
func (reconreqch ReconnectionProvider[Tm, PTm]) serveReconnects(ctx context.Context, ticktime time.Duration, targetbufsize int) { // TODO: test this
	buf := make([]*EpollReConnector[Tm, PTm], 0, targetbufsize)
	ticker := time.NewTicker(ticktime)
	dialer := &net.Dialer{Timeout: Reconnection_dial_timeout}

	for {
		select {
		case <-ctx.Done():
			return
		case req := <-reconreqch:
			buf = append(buf, req)
		case <-ticker.C:
			for i := 0; i < len(buf); i++ {
				buf[i].mux.Lock()
				if !buf[i].isstopped {
					if buf[i].connector.IsClosed() {
						conn, err := dialer.Dial(buf[i].reconAddr.netw, buf[i].reconAddr.address)
						if err != nil {
							buf[i].mux.Unlock()
							continue // не логается
						}
						if buf[i].doOnDial != nil {
							if err := buf[i].doOnDial(conn); err != nil {
								conn.Close()
								buf[i].mux.Unlock()
								continue
							}
						}
						conn.SetReadDeadline(time.Time{}) //TODO: нужно ли обнулять conn.readtimeout после doOnDial() ??

						newcon, err := NewEpollConnector[Tm, PTm](conn, buf[i])
						if err != nil {
							conn.Close()
							buf[i].mux.Unlock()
							continue // не логается
						}

						if err = newcon.StartServing(); err != nil {
							newcon.ClearFromCache()
							conn.Close()
							buf[i].mux.Unlock()
							continue // не логается
						}

						buf[i].connector = newcon

						if buf[i].doAfterReconnect != nil {
							if err := buf[i].doAfterReconnect(); err != nil {
								buf[i].connector.Close(err)
								buf[i].mux.Unlock()
								continue
							}
						}
					}
				}
				buf[i].mux.Unlock()

				buf = buf[:i+copy(buf[i:], buf[i+1:])] // трем из буфера

				i--
			}
			if cap(buf) > targetbufsize && len(buf) <= targetbufsize { // при переполнении буфера снова его уменьшаем, если к этому моменту разберемся с реконнектами // защиту от переполнения буфера ставить нельзя, иначе куда оверфловнутые реконнекты пихать
				newbuf := make([]*EpollReConnector[Tm, PTm], len(buf), targetbufsize)
				copy(newbuf, buf)
				buf = newbuf
			}
		}
	}
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
		reconnector.reconReqCh <- reconnector
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
