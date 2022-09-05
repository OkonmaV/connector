package connector

// import (
// 	"context"
// 	"errors"
// 	"net"
// 	"strings"
// 	"time"

// 	"github.com/big-larry/suckhttp"
// )

// type HttpMessage struct {
// 	Request *suckhttp.Request
// }

// // пргументом должен быть ConnReader, а не голый коннекшн, но suckhttp не разрешает, шоделатт?
// func (msg *HttpMessage) Read(conn net.Conn) (err error) {
// 	msg.Request, err = suckhttp.ReadRequest(context.Background(), conn, time.Minute) // TODO: context
// 	if err != nil {
// 		if strings.Contains(err.Error(), "Canceled") { // suckhttp/reader.go/line 42: errors.New("Canceled")
// 			return ErrReadTimeout
// 		}
// 		return
// 	}
// 	if msg.Request.GetHeader("x-request-id") == "" {
// 		return errors.New("not set x-request-id")
// 	}

// 	return nil
// }

// // nothing is allocated
// func NewHttpMessage() *HttpMessage {
// 	return &HttpMessage{}
// }

// // короч поля респонса в привате танцуют - нужно в suckhttp метод перегона в []byte (.Byte) добавить, ибо если насухую не со структурой suckhttp.Response
// // и suckhttp.Request в хэндлерах работать нужно будет, то это жопа
// func FormatResponse(response suckhttp.Response) ([]byte, error) { // TODO: и удалить эти форматтеры отсюда
// 	return suckhttp.CreateResponseMessage(response.S)
// }

// func FormatRequest(request suckhttp.Request) ([]byte, error) {
// 	request.String()
// 	return
// }
