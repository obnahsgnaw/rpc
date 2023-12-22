package rpcserver

import (
	"context"
	"errors"
	"github.com/obnahsgnaw/application/pkg/utils"
	"github.com/obnahsgnaw/http/listener"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"strconv"
)

type Server struct {
	server             *grpc.Server
	listener           *listener.PortedListener
	logger             *zap.Logger
	beforeInterceptors []func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) error
	afterHandlers      []func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, resp interface{}, err error)
	services           []rpcService
}

type rpcService struct {
	desc grpc.ServiceDesc
	serv interface{}
}

func New(lr *listener.PortedListener, l *zap.Logger) *Server {
	s := &Server{
		listener: lr,
		server:   nil,
		logger:   l,
	}
	s.server = grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer utils.RecoverHandler("handle", func(err1, stack string) {
			err = errors.New("handle failed, err=" + err1)
			if s.logger != nil {
				s.logger.Error("handle failed, err=" + err1 + ", stack=" + stack)
			}
		})
		for _, h := range s.beforeInterceptors {
			if err = h(ctx, req, info); err != nil {
				return
			}
		}
		resp, err = handler(ctx, req)
		for _, h := range s.afterHandlers {
			h(ctx, req, info, resp, err)
		}
		return
	}))
	return s
}

func (s *Server) Register(desc *grpc.ServiceDesc, serv interface{}) {
	s.services = append(s.services, rpcService{desc: *desc, serv: serv})
}

func (s *Server) Listener() *listener.PortedListener {
	return s.listener
}

func (s *Server) RegisterBeforeInterceptor(i func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) error) {
	if i != nil {
		s.beforeInterceptors = append(s.beforeInterceptors)
	}
}

func (s *Server) RegisterAfterHandler(h func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, resp interface{}, err error)) {
	if h != nil {
		s.afterHandlers = append(s.afterHandlers, h)
	}
}

func (s *Server) Start() error {
	s.init()
	return s.server.Serve(s.listener.Listener())
}

func (s *Server) SyncStart(cb func(err error)) {
	go func(rs *Server) {
		defer rs.Close()
		if err := rs.Start(); err != nil {
			cb(err)
			return
		}
	}(s)
}

func (s *Server) Close() {
	if s.server != nil {
		s.server.Stop()
	}

	log.Println("Server closed.")
}

func (s *Server) Addr() string {
	return "tcp:" + strconv.Itoa(s.listener.Port())
}

func (s *Server) Port() int {
	return s.listener.Port()
}

func (s *Server) init() {
	for _, h := range s.services {
		s.server.RegisterService(&h.desc, h.serv)
	}
}
