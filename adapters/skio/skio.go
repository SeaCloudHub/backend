package skio

import (
	"net"
	"net/url"

	_const "github.com/SeaCloudHub/backend/pkg/const"
)

type Conn interface {
	ID() string
	Close() error
	URL() url.URL
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	RemoteHeader() net.Addr

	Context() interface{}
	SetContext(v interface{})
	Namespace() string
	Emit(msg string, v ...interface{})
}

type AppSocket interface {
	Conn
	_const.Requester
}

type appSocket struct {
	Conn
	_const.Requester
}

func NewAppSocket(conn Conn, requester _const.Requester) *appSocket {
	return &appSocket{Conn: conn, Requester: requester}
}
