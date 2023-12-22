package rpc

import "github.com/obnahsgnaw/application/servertype"

type PServer struct {
	id string
	st servertype.ServerType
}

func NewPServer(id string, st servertype.ServerType) *PServer {
	return &PServer{
		id: id,
		st: st,
	}
}

func (s *PServer) Id() string {
	return s.id
}

func (s *PServer) ServerType() servertype.ServerType {
	return s.st
}
