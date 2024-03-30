package rpc

type Option func(s *Server)

func RegEnable() Option {
	return func(s *Server) {
		s.regAble = true
	}
}

func IgLrClose(ig bool) Option {
	return func(s *Server) {
		s.lrIgClose = ig
	}
}

func IgLrServe(ig bool) Option {
	return func(s *Server) {
		s.lrIgServe = ig
	}
}
