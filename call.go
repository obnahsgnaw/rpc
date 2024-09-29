package rpc

import (
	"context"
	"errors"
	"github.com/obnahsgnaw/rpc/pkg/rpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"time"
)

type RpsError struct {
	err error
}

func (e *RpsError) Error() string {
	return e.err.Error()
}

func newRpsError(msg string) *RpsError {
	return &RpsError{err: errors.New(msg)}
}

func (s *Server) Call(from, to, rqId, appid, uid string, cb func(context.Context, *grpc.ClientConn) error) error {
	m := rpcclient.Module(to)
	addr := s.Manager().GetRand(m)
	if addr == "" {
		return newRpsError("no rpc addr")
	}
	cc, err := s.Manager().GetConn(m, addr, 1)
	if err != nil {
		return newRpsError("fetch client failed")
	}

	if cb == nil {
		return newRpsError("callback is nil")
	}
	ctx, cl := context.WithTimeout(s.app.Context(), s.callTtl)
	defer cl()

	ctx = metadata.AppendToOutgoingContext(ctx, "app_id", appid)
	ctx = metadata.AppendToOutgoingContext(ctx, "user_id", uid)
	ctx = metadata.AppendToOutgoingContext(ctx, "rq_id", rqId)
	ctx = metadata.AppendToOutgoingContext(ctx, "rq_type", "rpc")
	ctx = metadata.AppendToOutgoingContext(ctx, "rq_from", from)
	ctx = metadata.AppendToOutgoingContext(ctx, "rq_to", to)

	if err = cb(ctx, cc); err != nil {
		var rqType string
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			rqIds := md.Get("rq_type")
			if len(rqIds) > 0 {
				rqType = rqIds[0]
			}
		}
		if rqType == "" {
			err = newRpsError(err.Error())
		}
	}
	return err
}

func (s *Server) SetCallTtl(ttl time.Duration) {
	if ttl < 10 {
		ttl = time.Second * ttl
	}
	s.callTtl = ttl
}

func (s *Server) IsRpsError(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *RpsError
	return errors.As(err, &rpcErr)
}
