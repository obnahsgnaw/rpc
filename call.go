package rpc

import (
	"context"
	"github.com/obnahsgnaw/rpc/pkg/rpcclient"
	"google.golang.org/grpc"
	"time"
)

type RpsError rpcclient.RpsError

func (s *Server) Call(from, to, rqId, appid, uid string, cb func(context.Context, *grpc.ClientConn) error) error {
	return s.Manager().Call(s.app.Context(), from, to, rqId, appid, uid, cb)
}

func (s *Server) ValCall(from, to, rqId, appid, uid string, cb func(context.Context, *grpc.ClientConn) (interface{}, error)) (interface{}, error) {
	return s.Manager().ValCall(s.app.Context(), from, to, rqId, appid, uid, cb)
}

func (s *Server) SetCallTtl(ttl time.Duration) {
	s.Manager().SetCallTtl(ttl)
}

func (s *Server) IsRpsError(err error) bool {
	return s.Manager().IsRpsError(err)
}

func (s *Server) IsCustomError(err error) *rpcclient.CustomError {
	return s.Manager().IsCustomError(err)
}
