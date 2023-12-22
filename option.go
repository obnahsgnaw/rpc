package rpc

type Option func(s *Server)

func with(s *Server, options ...Option) {
	for _, o := range options {
		if o != nil {
			o(s)
		}
	}
}

func RegEnable() Option {
	return func(s *Server) {
		s.regAble = true
	}
}

func Parent(ps *PServer) Option {
	return func(s *Server) {
		s.pServer = ps
	}
}

func IgLrClose(ig bool) Option {
	return func(s *Server) {
		s.lrIgClose = ig
	}
}
