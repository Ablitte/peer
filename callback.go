package peer

type ConnectionCallBack interface {
	OnClosed(*Session)
	OnReceive(*Session, []byte) error
}
