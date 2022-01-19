package tcp

import (
	"errors"
	log "github.com/greywords/logger"
	"github.com/greywords/peer"
	"net"
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

type tcpConnection struct {
	peer.ConnectionIdentify
	p         *peer.SessionManager
	conn      *net.TCPConn
	send      chan []byte
	running   bool
	closeOnce *sync.Once
	closeCh   chan struct{}
}

func (t *tcpConnection) Peer() *peer.SessionManager {
	return t.p
}

func (t *tcpConnection) Send(msg []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrBrokenPipe
			log.Debug("Send recover panic:", e)
		}
	}()
	if !t.running {
		return ErrBrokenPipe
	}
	if len(t.send) >= maxSendBuffer {
		return ErrBufferPoolExceed
	}
	if len(msg) > maxFrameMessageLen {
		return ErrSendBufferLimit
	}
	t.send <- msg
	return nil
}

func (t *tcpConnection) Close() {
	t.conn.Close()
	t.running = false
}

func (t *tcpConnection) ID() int64 {
	return t.ConnectionIdentify.ID()
}

func (t *tcpConnection) RemoteAddr() string {
	if t.conn == nil {
		return ""
	}
	return t.conn.RemoteAddr().String()
}

func (t *tcpConnection) IsClosed() bool {
	return !t.running
}

func (w *tcpConnection) init() {
	go w.recvLoop()
	go w.sendLoop()
	w.running = true
}

func newConnection(conn *net.TCPConn, p *peer.SessionManager) *tcpConnection {
	wsc := &tcpConnection{
		p:         p,
		conn:      conn,
		send:      make(chan []byte, maxSendBuffer),
		closeOnce: &sync.Once{},
		closeCh:   make(chan struct{}),
	}
	wsc.init()
	return wsc
}
func (ws *tcpConnection) close() {
	ws.p.UnRegister <- ws.ID()
	ws.conn.Close()
	ws.running = false
	ws.closeOnce.Do(func() {
		close(ws.closeCh)
	})
}

// 接收循环
func (t *tcpConnection) recvLoop() {
	defer func() {
		log.Debug("recvLoop: closed: sessId=%d", t.ID())
		t.close()
	}()
	t.conn.SetReadDeadline(time.Now().Add(pongWait))
	for t.conn != nil {
		data := make([]byte, maxFrameMessageLen)
		_, err := t.conn.Read(data)
		if err != nil {
			break
		}
		t.p.ProcessMessage(t.ID(), data)
	}
}

func (t *tcpConnection) sendLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Debug("sendLoop: closed: tId= %d", t.ID())
		ticker.Stop()
		t.conn.Close()
		t.running = false
		close(t.send)

	}()
	for {
		select {
		case <-t.closeCh:
			return
		case msg := <-t.send:
			t.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if _, err := t.conn.Write(msg); err != nil {
				return
			}
		case <-ticker.C:
			t.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if _, err := t.conn.Write(nil); err != nil {
				return
			}
		}
	}
}
