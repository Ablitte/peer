package tcp

import (
	log "github.com/greywords/logger"
	"github.com/greywords/peer"
	"net"
	"time"
)

func init() {
	peer.RegisterAcceptor("tcp", new(tcpAcceptor))
}

type tcpAcceptor struct {
	addr string
	run  bool
	ls   *net.TCPListener
}

func (w *tcpAcceptor) init(addr string) {
	w.addr = addr
	w.run = true
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatal("resolve tcp addr failed:", err.Error())
	}
	w.ls, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatal("listen tcp addr failed:", err.Error())
		panic(err.Error())
	}
	log.Debug("listen tcp on %s success,waiting connect!", addr)
}
func (w *tcpAcceptor) Start(addr string, mgr *peer.SessionManager) error {
	w.init(addr)
	for w.run {
		w.ls.SetDeadline(time.Now().Add(time.Second * 10))
		conn, err := w.ls.AcceptTCP()
		if err != nil {
			netErr, ok := err.(net.Error)

			//超时，无视，继续接收下一个
			if ok && netErr.Timeout() && netErr.Temporary() {
				continue
			}
			//其他错误。停止
			log.Debug("accept error:", err.Error())
			w.run = false
		}
		mgr.Register <- peer.NewSession(newConnection(conn, mgr))
	}
	return nil
}

func (w *tcpAcceptor) Info() string {
	return w.addr
}

func (w *tcpAcceptor) Stop() {
	//todo 关闭服务器调用
	w.run = false
}
