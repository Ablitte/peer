package ws

import (
	"errors"
	"github.com/gorilla/websocket"
	log "github.com/greywords/logger"
	"github.com/greywords/peer"

	"sync"
	"time"
)

const (
	writeWait          = 20 * time.Second
	pongWait           = 60 * time.Second
	pingPeriod         = (pongWait * 9) / 10
	maxFrameMessageLen = 16 * 1024 //4 * 4096
	maxSendBuffer      = 16
)

var (
	ErrBrokenPipe       = errors.New("send to broken pipe")
	ErrBufferPoolExceed = errors.New("send buffer exceed")
	ErrSendBufferLimit  = errors.New("send buffer limit")
)

type wsConnection struct {
	peer.ConnectionIdentify
	p         *peer.SessionManager
	conn      *websocket.Conn
	send      chan []byte
	running   bool
	closeOnce *sync.Once
	closeCh   chan struct{}
}

//func (w *wsConnection) Raw() interface{} {
//	if w.conn == nil {
//		return nil
//	}
//	return w.conn
//}

func (w *wsConnection) Peer() *peer.SessionManager {
	return w.p
}

func (w *wsConnection) Send(msg []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrBrokenPipe
			log.Debug("Send recover panic:", e)
		}
	}()
	if !w.running {
		return ErrBrokenPipe
	}
	if len(w.send) >= maxSendBuffer {
		return ErrBufferPoolExceed
	}
	if len(msg) > maxFrameMessageLen {
		return ErrSendBufferLimit
	}
	w.send <- msg
	return nil
}

func (w *wsConnection) Close() {
	w.conn.Close()
	w.running = false
}

func (w *wsConnection) RemoteAddr() string {
	if w.conn == nil {
		return ""
	}
	return w.conn.RemoteAddr().String()
}

func (w *wsConnection) IsClosed() bool {
	return !w.running
}

func (w *wsConnection) init() {
	go w.recvLoop()
	go w.sendLoop()
	w.running = true
}

func newConnection(conn *websocket.Conn, p *peer.SessionManager) *wsConnection {
	wsc := &wsConnection{
		p:         p,
		conn:      conn,
		send:      make(chan []byte, maxSendBuffer),
		closeOnce: &sync.Once{},
		closeCh:   make(chan struct{}),
	}
	wsc.init()
	return wsc
}
func (ws *wsConnection) close() {
	ws.p.UnRegister <- ws.ID()
	ws.conn.Close()
	ws.running = false
	ws.closeOnce.Do(func() {
		close(ws.closeCh)
	})
}

// 接收循环
func (ws *wsConnection) recvLoop() {
	defer func() {
		log.Debug("recvLoop: closed: wsId=%d", ws.ID())
		ws.close()
	}()
	ws.conn.SetReadDeadline(time.Now().Add(pongWait))
	ws.conn.SetPongHandler(func(string) error { ws.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for ws.conn != nil {
		_, data, err := ws.conn.ReadMessage()
		if err != nil {
			break
		}
		ws.p.ProcessMessage(ws.ID(), data)
	}
}

func (ws *wsConnection) sendLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Debug("sendLoop: closed: wsId= %d", ws.ID())
		ticker.Stop()
		ws.conn.Close()
		ws.running = false
		close(ws.send)

	}()
	for {
		select {
		case <-ws.closeCh:
			return
		case msg := <-ws.send:
			ws.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			ws.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
