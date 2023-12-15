package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/obnahsgnaw/application/pkg/utils"
	"google.golang.org/grpc"
	"log"
	"net"
	"strconv"
)

type Server struct {
	listener       net.Listener
	s              *grpc.Server
	protocol       string
	port           int
	beforeHandlers []func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) error
	afterHandlers  []func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, resp interface{}, err error)
	services       []rpcService
}
type rpcService struct {
	desc grpc.ServiceDesc
	serv interface{}
}

// NewServer return rpc server instance
func NewServer(port int) *Server {
	return &Server{
		listener: nil,
		s:        nil,
		port:     port,
		protocol: "tcp",
	}
}

// Register register rpc server
func (rs *Server) Register(desc *grpc.ServiceDesc, serv interface{}) {
	rs.services = append(rs.services, rpcService{desc: *desc, serv: serv})
}

func (rs *Server) init() (err error) {
	if rs.listener, err = net.Listen("tcp", utils.ToStr(":", strconv.Itoa(rs.port))); err != nil {
		return err
	}

	rs.s = grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if err1 := recover(); err1 != nil {
				err = errors.New("rpc middleware error")
				fmt.Printf("%v \n", err1)
				return
			}
		}()
		for _, h := range rs.beforeHandlers {
			if err = h(ctx, req, info); err != nil {
				return
			}
		}
		resp, err = handler(ctx, req)
		for _, h := range rs.afterHandlers {
			h(ctx, req, info, resp, err)
		}
		return
	}))

	for _, h := range rs.services {
		rs.s.RegisterService(&h.desc, h.serv)
	}

	return nil
}

func (rs *Server) RegisterBeforeInterceptor(i func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) error) {
	if i != nil {
		rs.beforeHandlers = append(rs.beforeHandlers)
	}
}

func (rs *Server) RegisterAfterHandler(h func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, resp interface{}, err error)) {
	if h != nil {
		rs.afterHandlers = append(rs.afterHandlers, h)
	}
}

// Start rpc server
func (rs *Server) Start() error {
	if err := rs.init(); err != nil {
		return err
	}
	return rs.s.Serve(rs.listener)
}

// Close rpc server
func (rs *Server) Close() {
	if rs.s != nil {
		rs.s.Stop()
		_ = rs.listener.Close()
	}

	log.Println("Server closed.")
}

// Addr return rpc addr
func (rs *Server) Addr() string {
	return fmt.Sprintf("%s:%d", rs.protocol, rs.port)
}

// Port return rpc port
func (rs *Server) Port() int {
	return rs.port
}

// SyncStart sync start
func (rs *Server) SyncStart(cb func(err error)) {
	go func(rs *Server) {
		defer rs.Close()
		if err := rs.Start(); err != nil {
			cb(err)
			return
		}
	}(rs)
}
