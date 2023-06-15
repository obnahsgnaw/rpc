package rpc

import (
	"errors"
	"github.com/obnahsgnaw/application/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"sync"
	"time"
)

type Addr map[string]map[int]*grpc.ClientConn

func (a Addr) Add(server string) {
	a[server] = make(map[int]*grpc.ClientConn)
}

type Module string

func (m Module) String() string {
	return string(m)
}

// Manager rpc server addr manager
type Manager struct {
	sync.Mutex
	addrMap map[Module]Addr
}

// NewManager return a new addr manager
func NewManager() *Manager {
	return &Manager{addrMap: make(map[Module]Addr)}
}

// Add add a module server addr
func (m *Manager) Add(module Module, addr string) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.addrMap[module]; !ok {
		m.addrMap[module] = make(Addr)
	}
	m.addrMap[module].Add(addr)
}

// Rm remove a module server addr
func (m *Manager) Rm(module Module, addr string) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.addrMap[module]; ok {
		if _, ok = m.addrMap[module][addr]; ok {
			if m.addrMap[module][addr] != nil {
				for _, c := range m.addrMap[module][addr] {
					_ = c.Close()
				}
			}
			delete(m.addrMap[module], addr)
		}
	}
}

// Get return module server addr list
func (m *Manager) Get(module Module) (addrList []string) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.addrMap[module]; ok {
		for k := range m.addrMap[module] {
			addrList = append(addrList, k)
		}
	}
	return
}

// GetRand return one module server addr or empty if not
func (m *Manager) GetRand(module Module) string {
	list := m.Get(module)
	if len(list) > 0 {
		return list[utils.RandInt(len(list))]
	}
	return ""
}

// GetConn return rpc conn
func (m *Manager) GetConn(module Module, addr string, tag int) (*grpc.ClientConn, error) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.addrMap[module]; ok {
		if _, ok = m.addrMap[module][addr]; ok {
			if m.addrMap[module][addr][tag] == nil {
				c, err := m.newClient(addr)
				if err != nil {
					return nil, err
				}
				m.addrMap[module][addr][tag] = c
			}
			return m.addrMap[module][addr][tag], nil
		}
	}

	return nil, errors.New("not found")
}

func (m *Manager) newClient(server string) (*grpc.ClientConn, error) {
	return grpc.Dial(
		server,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                100 * time.Second,
				Timeout:             20 * time.Second,
				PermitWithoutStream: true,
			}),
	)
}

// Release all rpc client
func (m *Manager) Release() {
	for _, c := range m.addrMap {
		for _, cc := range c {
			for _, ccc := range cc {
				_ = ccc.Close()
			}
		}
	}
}
