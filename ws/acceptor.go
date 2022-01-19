package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/greywords/logger"
	"github.com/greywords/peer"

	"net/http"
	"net/url"
)

func init() {
	peer.RegisterAcceptor("ws", new(wsAcceptor))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type wsAcceptor struct {
	addr string
}

func (w *wsAcceptor) Start(addr string, mgr *peer.SessionManager) error {
	w.addr = addr
	urlObj, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("websocket urlparse failed. url(%s) %v", addr, err)
	}
	if urlObj.Path == "" {
		return fmt.Errorf("websocket start failed. expect path in url to listen addr:%s", addr)
	}
	http.HandleFunc(urlObj.Path, func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Debug(err.Error())
			return
		}
		log.Debug("accept client from=%s", c.RemoteAddr().String())
		mgr.Register <- peer.NewSession(newConnection(c, mgr))
	})
	log.Debug("listen websocket on %s success,waiting connect!", addr)

	err = http.ListenAndServe(urlObj.Host, nil)
	if err != nil {
		return fmt.Errorf("websocket ListenAndServe addr:%s failed:%v", addr, err)
	}
	return nil
}

func (w *wsAcceptor) Info() string {
	return w.addr
}

func (w *wsAcceptor) Stop() {
	//todo 关闭服务器调用
}
