package rpc

import (
	"io"
	"log"
)

type Option func(s *Server)

func RegEnable() Option {
	return func(s *Server) {
		s.regAble = true
	}
}

func AccessWriter(w io.Writer) Option {
	return func(s *Server) {
		s.accessWriter = w
	}
}
func ErrorWriter(w io.Writer) Option {
	return func(s *Server) {
		if w != nil {
			s.errLogger = log.New(w, "\n\n\x1b[31m", log.LstdFlags)
		}
	}
}
