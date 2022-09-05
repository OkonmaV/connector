package connector

// type Connector struct {
// 	conn       net.Conn
// 	msghandler MessageHandler
// 	mux        sync.RWMutex
// 	isclosed   bool
// }

// func NewConnector(conn net.Conn, messagehandler MessageHandler) (*Connector, error) {
// 	if conn == nil {
// 		return nil, ErrNilConn
// 	}
// 	if pool == nil {
// 		panic(ErrNilGopool)
// 	}

// 	connector := &Connector{conn: conn, msghandler: messagehandler}

// 	return connector, nil
// }

// func (connector *Connector) StartServing(sheduletimeout time.Duration, keepAlive bool) error {
// 	//TODO: second serving???
// 	return pool.ScheduleWithTimeout(func() { connector.handle(keepAlive) }, sheduletimeout)
// }

// func (connector *Connector) handle(keepAlive bool) {
// 	for {
// 		message := connector.msghandler.NewMessage()

// 		err := message.Read(connector.conn)
// 		if err != nil {
// 			if keepAlive && errors.Is(err, ErrReadTimeout) {
// 				continue
// 			}
// 			connector.Close(err)
// 			return
// 		}
// 		if err = connector.msghandler.Handle(message); err != nil {
// 			connector.Close(err)
// 			return
// 		}
// 		if !keepAlive {
// 			connector.Close(nil)
// 			return
// 		}
// 	}
// }

// func (connector *Connector) Send(message []byte) error {

// 	if connector.IsClosed() {
// 		return ErrClosedConnector
// 	}
// 	//connector.conn.SetWriteDeadline(time.Now().Add(time.Second))
// 	_, err := connector.conn.Write(message)
// 	return err
// }

// func (connector *Connector) Close(reason error) {
// 	connector.mux.Lock()
// 	defer connector.mux.Unlock()

// 	if connector.isclosed {
// 		return
// 	}
// 	connector.isclosed = true
// 	connector.conn.Close()
// 	connector.msghandler.HandleClose(reason)
// }

// // call in HandleClose() will cause deadlock
// func (connector *Connector) IsClosed() bool {
// 	return connector.isclosed
// }

// func (connector *Connector) RemoteAddr() net.Addr {
// 	return connector.conn.RemoteAddr()
// }
