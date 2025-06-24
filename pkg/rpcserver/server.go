package rpcserver

import (
	"context"
	"errors"
	"github.com/obnahsgnaw/application/pkg/utils"
	"github.com/obnahsgnaw/http/listener"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strconv"
	"sync"
)

type BeforeInterceptor func(ctx context.Context, head Header, req interface{}, info *grpc.UnaryServerInfo) error

type AfterHandler func(ctx context.Context, head Header, req interface{}, info *grpc.UnaryServerInfo, resp interface{}, err error)

type Server struct {
	lc                 sync.Mutex
	server             *grpc.Server
	listener           *listener.PortedListener
	logger             *zap.Logger
	beforeInterceptors []BeforeInterceptor
	afterHandlers      []AfterHandler
	services           []rpcService
	startKey           string
	errParser          func(err error) (code string, message string, statusCode string)
}

type Header struct {
	RqId   string
	From   string
	To     string
	AppId  string
	UserId string
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
		defer func() {
			if err != nil {
				var code = "1"
				var statusCode = "500"
				var message = err.Error()
				if s.errParser != nil {
					code, message, statusCode = s.errParser(err)
				}
				err = grpc.SetHeader(ctx, metadata.New(map[string]string{
					"err_code":    code,
					"err_message": message,
					"err_status":  statusCode,
				}))
				err = nil
			}
		}()
		head := s.parseHeader(ctx)
		for _, h := range s.beforeInterceptors {
			if err = h(ctx, head, req, info); err != nil {
				return
			}
		}
		resp, err = handler(ctx, req)
		for _, h := range s.afterHandlers {
			h(ctx, head, req, info, resp, err)
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

func (s *Server) RegisterBeforeInterceptor(i BeforeInterceptor) {
	if i != nil {
		s.beforeInterceptors = append(s.beforeInterceptors)
	}
}

func (s *Server) RegisterAfterHandler(h AfterHandler) {
	if h != nil {
		s.afterHandlers = append(s.afterHandlers, h)
	}
}

func (s *Server) Start(key string) error {
	s.lc.Lock()
	defer s.lc.Unlock()
	if s.startKey != "" {
		return nil
	}
	s.startKey = key
	s.init()
	l := s.listener.GrpcListener()
	l = newNoCl(l)
	err := s.server.Serve(l)
	if err != nil {
		s.startKey = ""
	}
	return err
}

func (s *Server) SyncStart(key string, cb func(err error)) {
	go func(rs *Server) {
		defer rs.Close(key)
		if err := rs.Start(key); err != nil {
			cb(err)
			return
		}
	}(s)
}

func (s *Server) Close(key string) {
	if key != s.startKey {
		return
	}
	if s.server != nil {
		s.server.GracefulStop()
	}
	s.startKey = ""
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

func (s *Server) parseHeader(ctx context.Context) Header {
	var rqId, rqFrom, rqTo, appId, userId string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		rqIds := md.Get("rq_id")
		if len(rqIds) > 0 {
			rqId = rqIds[0]
		}
		rqFroms := md.Get("rq_from")
		if len(rqFroms) > 0 {
			rqFrom = rqFroms[0]
		}
		rqTos := md.Get("rq_to")
		if len(rqTos) > 0 {
			rqTo = rqTos[0]
		}
		appIds := md.Get("app_id")
		if len(appIds) > 0 {
			appId = appIds[0]
		}
		userIds := md.Get("user_id")
		if len(userIds) > 0 {
			userId = userIds[0]
		}
	}
	return Header{
		RqId:   rqId,
		From:   rqFrom,
		To:     rqTo,
		AppId:  appId,
		UserId: userId,
	}
}

func (s *Server) SetCustomErrorParser(f func(err error) (code string, message string, statusCode string)) {
	s.errParser = f
}
