# connector

## Usage

```go
package main

import (
	"context"
	"net"
	"connector"
	"dynamicworkerspool"
	"time"
)

//connector.Readable interface implementation:
type message struct{}

func (*Message)Read(net.Conn) error {return nil}

//connector.MessageHandler[message] interface implementation:
type foo struct {
    con *connector.EpollConnector[message, *message]
    recon *connector.EpollConnector[message, *message]
}

func (*foo) Handle(*message) error {return nil}

func (*foo) HandleClose(error) {}

//start do some shit:
func main(){
    connector.SetupEpoll(nil)
    connector.SetupPoolHandling(dynamicworkerspool.NewPool(2, 5, time.Second))

    f:=&foo{}
    conn,_ := net.Dial("tcp","127.0.0.1:8080")


    //connector:
    con, _ := connector.NewEpollConnector[message](conn, f)
    err := con.StartServing(); err != nil {
        con.ClearFromCache()
        return
    }
    con.Close(nil)

    //reconnector:
    connector.SetupReconnection(context.Background(), time.Second, 1, 1)
    recon, _ := connector.NewEpollReConnector[message](conn, f, nil, nil)
    err := recon.StartServing(); err != nil {
        recon.ClearFromCache()
        return
    }
    recon.Close(nil)
}
```

