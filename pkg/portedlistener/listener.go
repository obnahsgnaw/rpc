package portedlistener

import (
	"github.com/obnahsgnaw/application/pkg/url"
	"net"
)

type PortedListener struct {
	l net.Listener
	h url.Host
}

func New(l net.Listener, host url.Host) *PortedListener {
	return &PortedListener{
		l: l,
		h: host,
	}
}

func (s *PortedListener) Listener() net.Listener {
	return s.l
}

func (s *PortedListener) Host() url.Host {
	return s.h
}

func (s *PortedListener) Ip() string {
	return s.h.Ip
}

func (s *PortedListener) Port() int {
	return s.h.Port
}
