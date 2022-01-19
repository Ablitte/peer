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
	w.closeOnce.Do(func() {
		close(w.closeCh)
	})
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
func (w *wsConnection) close() {
	w.p.UnRegister <- w.ID()
	w.conn.Close()
	w.running = false
	w.closeOnce.Do(func() {
		close(w.closeCh)
	})
}

// 接收循环
func (w *wsConnection) recvLoop() {
	defer func() {
		log.Debug("recvLoop: closed: wsId=%d", w.ID())
		w.close()
	}()
	w.conn.SetReadDeadline(time.Now().Add(pongWait))
	w.conn.SetPongHandler(func(string) error { w.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for w.conn != nil {
		_, data, err := w.conn.ReadMessage()
		if err != nil {
			break
		}
		w.p.ProcessMessage(w.ID(), data)
	}
}

func (w *wsConnection) sendLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Debug("sendLoop: closed: wsId= %d", w.ID())
		ticker.Stop()
		w.conn.Close()
		w.running = false
		close(w.send)

	}()
	for {
		select {
		case <-w.closeCh:
			return
		case msg := <-w.send:
			w.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := w.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			w.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := w.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
