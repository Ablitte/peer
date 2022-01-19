package peer

import "sync"

type Session struct {
	Conn       Connection
	data       map[string]interface{}
	rwDataLock sync.RWMutex
}

//设置
func (s *Session) Set(key string, value interface{}) {
	s.rwDataLock.Lock()
	defer s.rwDataLock.Unlock()
	s.data[key] = value
}

//取值
func (s *Session) Get(key string) (interface{}, bool) {
	s.rwDataLock.RLock()
	defer s.rwDataLock.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

//删除
func (s *Session) Delete(key string) {
	s.rwDataLock.Lock()
	defer s.rwDataLock.Unlock()
	delete(s.data, key)
}

func NewSession(conn Connection) *Session {
	s := &Session{
		Conn: conn,
		data: make(map[string]interface{}),
	}
	return s
}
