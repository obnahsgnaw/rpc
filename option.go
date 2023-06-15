package rpc

import "github.com/obnahsgnaw/application"

type Option func(s *Server)

func RegEnable() Option {
	return func(s *Server) {
		s.regEnable = true
	}
}
func Parent(p application.Server) Option {
	return func(s *Server) {
		s.pServer = p
		s.regInfo.ServerInfo.Type = p.Type().String()
	}
}
