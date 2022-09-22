package connector

import (
	"errors"
	"net"
	"sync"
	"time"
)

var ErrConnsLimitReached error = errors.New("reached conns limit")

// Количество поднятых на чтение горутин (= незакрытых коннекторов) неконтролируемо (= либе до пизды),
// контролировать должен тот, кто эти коннекторы создает.
// Чтобы завершить все горутины, создатель коннектора должен сам закрыть все коннекторы (= либе до пизды),
// чтобы проконтроллировать завершение всех горутин описаны методы Wait.

type Connector[Tm any, PTm interface {
	Readable
	*Tm
}] struct {
	conn       net.Conn
	msghandler MessageHandler[PTm]
	mux        sync.Mutex
	isclosed   bool
}

var wg sync.WaitGroup

func NewConnector[Tmessage any,
	PTmessage interface {
		Readable
		*Tmessage
	}, Th MessageHandler[PTmessage]](conn net.Conn, messagehandler Th) (*Connector[Tmessage, PTmessage], error) {
	if conn == nil {
		return nil, ErrNilConn
	}
	if pool == nil {
		panic(ErrNilGopool)
	}

	connector := &Connector[Tmessage, PTmessage]{conn: conn, msghandler: messagehandler}

	return connector, nil
}

func (connector *Connector[Tm, PTm]) StartServing( /* available_worker_timeout time.Duration */ ) error {
	// if available_worker_timeout == 0 {
	// 	select {
	// 	case non_e_workers <- struct{}{}:
	// 		break
	// 	default:
	// 		return ErrConnsLimitReached
	// 	}
	// } else {
	// 	tmr := time.NewTimer(available_worker_timeout)
	// 	select {
	// 	case non_e_workers <- struct{}{}:
	// 		tmr.Stop()
	// 		break
	// 	case <-tmr.C:
	// 		return ErrConnsLimitReached
	// 	}
	// }

	wg.Add(1)
	go func() {
		var err error
		connector.conn.SetReadDeadline(time.Time{})

		for {
			message := PTm(new(Tm))
			if err = message.ReadWithoutDeadline(connector.conn); err != nil {
				break
			}
			if pool != nil {
				pool.Schedule(func() {
					if err := connector.msghandler.Handle(message); err != nil {
						connector.Close(err) //вызовет ошибку на чтении = разрыв петли
					}
				})
				continue
			}
			if err = connector.msghandler.Handle(message); err != nil {
				break
			}
		}
		connector.Close(err)
		wg.Done()
		//<-non_e_workers
	}()
	return nil
}

func (connector *Connector[_, _]) Send(message []byte) error {

	if connector.IsClosed() {
		return ErrClosedConnector
	}
	//connector.conn.SetWriteDeadline(time.Now().Add(time.Second))
	_, err := connector.conn.Write(message)
	return err
}

func (connector *Connector[_, _]) Close(reason error) {
	connector.mux.Lock()
	defer connector.mux.Unlock()

	if connector.isclosed {
		return
	}
	connector.isclosed = true
	connector.conn.Close()
	connector.msghandler.HandleClose(reason)
}

// call in HandleClose() will cause deadlock
func (connector *Connector[_, _]) IsClosed() bool {
	connector.mux.Lock()
	defer connector.mux.Unlock()
	return connector.isclosed
}

func (connector *Connector[_, _]) RemoteAddr() net.Addr {
	return connector.conn.RemoteAddr()
}

func WaitAllNonEpollWorkersDone() {
	wg.Wait()
}

func WaitAllNonEpollWorkersDoneWithTimeout(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	select {
	case <-done:
		return nil
	case <-timer.C:
		return errors.New("timeout reached")
	}
}
