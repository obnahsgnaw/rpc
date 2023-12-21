package rpc

import (
	"errors"
	"github.com/obnahsgnaw/application"
	"github.com/obnahsgnaw/application/pkg/url"
	"github.com/obnahsgnaw/rpc/pkg/portedlistener"
)

type Option func(s *Server)

func RegEnable() Option {
	return func(s *Server) {
		s.regEnable = true
	}
}
func Parent(p application.Server) Option {
	return func(s *Server) {
		s.pServer = p
		if _, ok := s.regInfos[s.id]; ok {
			s.regInfos[s.id].ServerInfo.Type = p.Type().String()
		}
	}
}
func Host(host url.Host) Option {
	return func(s *Server) {
		s.host = host
	}
}

func Listener(l *portedlistener.PortedListener) Option {
	return func(s *Server) {
		if l == nil {
			s.addErr(errors.New("listener is nil"))
		}
		s.listener = l.Listener()
		s.host = l.Host()
	}
}
