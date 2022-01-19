package peer

type Connection interface {
	// Peer 获得管理
	Peer() *SessionManager
	// Send 发送消息
	Send(msg []byte) error
	// Close 断开
	Close()
	// ID 标示ID
	ID() int64
	// RemoteAddr 访问地址
	RemoteAddr() string

	IsClosed() bool
}

type ConnectionIdentify struct {
	id int64
}

func (ci *ConnectionIdentify) ID() int64 {
	return ci.id
}

func (ci *ConnectionIdentify) SetID(id int64) {
	ci.id = id
}
