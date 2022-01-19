package peer

import log "github.com/greywords/logger"

type Acceptor interface {
	Start(name string, mgr *SessionManager) error
	Info() string
	Stop()
}

var acceptorList = make(map[string]Acceptor)

func RegisterAcceptor(name string, acceptor Acceptor) {
	if acceptor == nil {
		panic("acceptor: Register provide is nil")
	}
	if _, dup := acceptorList[name]; dup {
		panic("acceptor: Register called twice for provide " + name)
	}
	acceptorList[name] = acceptor
	log.Debug("Register %s Acceptor Success", acceptor.Info())
}

func GetAcceptor(name string) Acceptor {
	if v, ok := acceptorList[name]; ok {
		return v
	}
	return nil
}
