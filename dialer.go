package peer

type Dialer interface {
	DialServer(name string) *Session
}
