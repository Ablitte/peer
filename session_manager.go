package peer

import (
	"sync"
	"sync/atomic"
)

type SessionManager struct {
	list       sync.Map
	connIDGen  int64
	count      int64
	callback   ConnectionCallBack
	Register   chan *Session
	UnRegister chan int64
}

func (mgr *SessionManager) SetIDBase(base int64) {
	atomic.StoreInt64(&mgr.connIDGen, base)
}

func (mgr *SessionManager) Count() int {
	return int(atomic.LoadInt64(&mgr.count))
}

//新的连接
func (mgr *SessionManager) Add(sess *Session) {

	id := atomic.AddInt64(&mgr.connIDGen, 1)

	atomic.AddInt64(&mgr.count, 1)

	sess.Conn.(interface {
		SetID(int64)
	}).SetID(id)

	mgr.list.Store(id, sess)
}

//连接关闭
func (mgr *SessionManager) Close(id int64) {
	if v, ok := mgr.list.Load(id); ok {
		if mgr.callback != nil {
			go mgr.callback.OnClosed(v.(*Session))
		}
	}
	mgr.list.Delete(id)
	atomic.AddInt64(&mgr.count, -1)
}

//消息处理
func (mgr *SessionManager) ProcessMessage(sess int64, msg []byte) {

	if v, ok := mgr.list.Load(sess); ok {
		if mgr.callback != nil {
			if err := mgr.callback.OnReceive(v.(*Session), msg); err != nil {
				v.(*Session).Conn.Close()
			}
		}
	}
}
func (mgr *SessionManager) run() {
	for {
		select {
		case client := <-mgr.Register:
			mgr.connIDGen++
			mgr.count++
			client.Conn.(interface {
				SetID(int64)
			}).SetID(mgr.connIDGen)
			mgr.list.Store(mgr.connIDGen, client)
		case clientID := <-mgr.UnRegister:
			if v, ok := mgr.list.Load(clientID); ok {
				if mgr.callback != nil {
					mgr.callback.OnClosed(v.(*Session))
				}
			}
			mgr.list.Delete(clientID)
			mgr.count--
		}
	}
}

// 获得一个连接
func (mgr *SessionManager) GetSession(id int64) *Session {
	if v, ok := mgr.list.Load(id); ok {
		return v.(*Session)
	}
	return nil
}

func (mgr *SessionManager) VisitSession(callback func(*Session) bool) {
	mgr.list.Range(func(key, value interface{}) bool {
		return callback(value.(*Session))
	})
}

func (mgr *SessionManager) CloseAllSession() {
	mgr.VisitSession(func(sess *Session) bool {
		sess.Conn.Close()
		return true
	})
}

//连接数量
func (mgr *SessionManager) SessionCount() int64 {
	return atomic.LoadInt64(&mgr.count)
}

func NewSessionMgr(callback ConnectionCallBack) *SessionManager {
	s := &SessionManager{
		callback:   callback,
		Register:   make(chan *Session, 255),
		UnRegister: make(chan int64, 255),
	}
	go s.run()
	return s
}
