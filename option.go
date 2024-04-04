package rpc

type Option func(s *Server)

func RegEnable() Option {
	return func(s *Server) {
		s.regAble = true
	}
}
