package skio

import (
	"fmt"
	"sync"

	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

type RealtimeEngine interface {
	UserSockets(userId int) []AppSocket
	EmitToRoom(room string, key string, data interface{}) error
	EmitToUser(userId int, key string, data interface{}) error
}

type rtEngine struct {
	server  *socketio.Server
	storage map[int][]AppSocket
	locker  *sync.RWMutex
}

func NewEngine() *rtEngine {
	return &rtEngine{

		storage: make(map[int][]AppSocket),
		locker:  new(sync.RWMutex),
	}
}

func (engine *rtEngine) saveAppSocket(userId int, appSck AppSocket) {
	engine.locker.Lock()

	if v, ok := engine.storage[userId]; ok {
		engine.storage[userId] = append(v, appSck)
	} else {
		engine.storage[userId] = []AppSocket{appSck}
	}
}

func (engine *rtEngine) getAppSocket(userId int) []AppSocket {
	engine.locker.RLock()

	defer engine.locker.RUnlock()

	return engine.storage[userId]
}

func (engine *rtEngine) removeAppSocket(userId int, appSck AppSocket) {
	engine.locker.Lock()
	defer engine.locker.Unlock()

	if v, ok := engine.storage[userId]; ok {
		for i := range v {
			if v[i] == appSck {
				engine.storage[userId] = append(v[:i], v[i+1:]...)
				break
			}
		}
	}
}

func (engine *rtEngine) UserSockets(userId int) []AppSocket {
	var sockets []AppSocket

	if scks, ok := engine.storage[userId]; ok {
		return scks
	}

	return sockets
}

func (engine *rtEngine) EmitToRoom(room string, key string, data interface{}) error {
	engine.server.BroadcastToRoom("/", room, key, data)
	return nil
}

func (engine *rtEngine) EmitToUser(userId int, key string, data interface{}) error {
	sockets := engine.getAppSocket(userId)

	for _, s := range sockets {
		s.Emit(key, data)
	}

	return nil
}

func (engine *rtEngine) Run() error {
	server, err := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{websocket.Default},
	})

	if err != nil {
		return err
	}

	engine.server = server

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		return nil
	})

	server.OnError("/", func(c socketio.Conn, err error) {
		fmt.Errorf("rtEngine.Run.OnError: ", err)
	})

	server.OnDisconnect("/", func(c socketio.Conn, s string) {

	})

	server.OnEvent("/", "authenticate", func(s socketio.Conn, token string) {

	})

	return nil

}
