package rpc

import (
	"fmt"
	"github.com/obnahsgnaw/application/pkg/utils"
	"google.golang.org/grpc"
	"log"
	"net"
	"strconv"
)

type Server struct {
	listener net.Listener
	s        *grpc.Server
	protocol string
	port     int
}

// NewServer return rpc server instance
func NewServer(port int) (*Server, error) {
	listen, err := net.Listen("tcp", utils.ToStr(":", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}

	s := grpc.NewServer()

	return &Server{
		listener: listen,
		s:        s,
		port:     port,
		protocol: "tcp",
	}, nil
}

// Register register rpc server
func (rs *Server) Register(desc *grpc.ServiceDesc, serv interface{}) {
	rs.s.RegisterService(desc, serv)
}

// Start rpc server
func (rs *Server) Start() error {
	return rs.s.Serve(rs.listener)
}

// Close rpc server
func (rs *Server) Close() {
	rs.s.Stop()
	_ = rs.listener.Close()

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
