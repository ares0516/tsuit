package common

import (
	"sync"

	"github.com/hashicorp/yamux"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	sync.Mutex
	addr2session map[string]*yamux.Session
}

func NewManager() *Manager {
	return &Manager{
		addr2session: make(map[string]*yamux.Session),
	}
}

func (m *Manager) Add(addr string, session *yamux.Session) {
	m.Lock()
	defer m.Unlock()
	m.addr2session[addr] = session
}

func (m *Manager) Get(addr string) *yamux.Session {
	m.Lock()
	defer m.Unlock()
	session, ok := m.addr2session[addr]
	if !ok {
		return nil
	}
	return session
}

func (m *Manager) Remove(addr string) {
	m.Lock()
	defer m.Unlock()
	delete(m.addr2session, addr)
}

func (m *Manager) IsExist(addr string) bool {
	m.Lock()
	defer m.Unlock()
	_, ok := m.addr2session[addr]
	return ok
}

func (m *Manager) Dump() {
	m.Lock()
	defer m.Unlock()
	for addr, session := range m.addr2session {
		logrus.Infof("Dump Addr: %s, Session: %v", addr, session.RemoteAddr().String())
	}
}
