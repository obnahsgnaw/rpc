package rpcclient

import (
	"context"
	"errors"
	"github.com/obnahsgnaw/application/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"sync"
	"time"
)

type BeforeInterceptor func(ctx context.Context, head Header, method string, req interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error

type AfterHandler func(ctx context.Context, head Header, method string, req, reply interface{}, cc *grpc.ClientConn, err error, opts ...grpc.CallOption)

type Header struct {
	RqId   string
	From   string
	To     string
	AppId  string
	UserId string
}

// Manager rpc server addr manager
type Manager struct {
	sync.Mutex
	addrMap            map[Module]Addr
	beforeInterceptors []BeforeInterceptor
	afterHandlers      []AfterHandler
	callTtl            time.Duration
	errBuilder         func(code, message, statusCode string) error
}

type RpcMetadata struct {
	Header  metadata.MD
	Trailer metadata.MD
}

func newRpcMetadataContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, "RpcMetadata", &RpcMetadata{})
}

func getRpcMetadataContext(ctx context.Context) *RpcMetadata {
	return ctx.Value("RpcMetadata").(*RpcMetadata)
}

// NewManager return a new addr manager
func NewManager() *Manager {
	return &Manager{addrMap: make(map[Module]Addr), callTtl: time.Second * 3}
}

// Add add a module server addr
func (m *Manager) Add(module Module, addr string) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.addrMap[module]; !ok {
		m.addrMap[module] = make(Addr)
	}
	m.addrMap[module].Add(addr)
}

// Rm remove a module server addr
func (m *Manager) Rm(module Module, addr string) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.addrMap[module]; ok {
		if _, ok = m.addrMap[module][addr]; ok {
			if m.addrMap[module][addr] != nil {
				for _, c := range m.addrMap[module][addr] {
					_ = c.Close()
				}
			}
			delete(m.addrMap[module], addr)
		}
	}
}

// Get return module server addr list
func (m *Manager) Get(module Module) (addrList []string) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.addrMap[module]; ok {
		for k := range m.addrMap[module] {
			addrList = append(addrList, k)
		}
	}
	return
}

// GetRand return one module server addr or empty if not
func (m *Manager) GetRand(module Module) string {
	list := m.Get(module)
	if len(list) > 0 {
		return list[utils.RandInt(len(list))]
	}
	return ""
}

// GetConn return rpc conn
func (m *Manager) GetConn(module Module, addr string, tag int) (*grpc.ClientConn, error) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.addrMap[module]; ok {
		if _, ok = m.addrMap[module][addr]; ok {
			if m.addrMap[module][addr][tag] == nil {
				c, err := m.newClient(addr)
				if err != nil {
					return nil, err
				}
				m.addrMap[module][addr][tag] = c
			}
			return m.addrMap[module][addr][tag], nil
		}
	}

	return nil, errors.New("not found")
}

func (m *Manager) RegisterBeforeInterceptor(interceptor BeforeInterceptor) {
	m.beforeInterceptors = append(m.beforeInterceptors, interceptor)
}

func (m *Manager) RegisterAfterHandler(h AfterHandler) {
	m.afterHandlers = append(m.afterHandlers, h)
}

func (m *Manager) newClient(server string) (*grpc.ClientConn, error) {
	return grpc.Dial(
		server,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                100 * time.Second,
				Timeout:             20 * time.Second,
				PermitWithoutStream: true,
			},
		),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
			var mt *RpcMetadata
			defer func() {
				if err == nil {
					errCode := "1"
					errStatus := "500"
					errMessage := ""
					if mt != nil {
						errMessages := mt.Header.Get("err_message")
						if len(errMessages) > 0 {
							errMessage = errMessages[0]
						}
						errCodes := mt.Header.Get("err_code")
						if len(errCodes) > 0 {
							errCode = errCodes[0]
						}
						errStatuss := mt.Header.Get("err_status")
						if len(errStatuss) > 0 {
							errStatus = errStatuss[0]
						}
					}
					if errMessage != "" {
						if m.errBuilder != nil {
							err = m.errBuilder(errCode, errMessage, errStatus)
						} else {
							err = errors.New(errMessage + "[" + errStatus + " " + errCode + "]")
						}
						err = NewCustomError(err)
					}
				}
			}()
			header := m.parseHeader(ctx)
			for _, h := range m.beforeInterceptors {
				if err = h(ctx, header, method, req, cc, opts...); err != nil {
					return
				}
			}
			mt = getRpcMetadataContext(ctx)
			opts = append(opts, grpc.Header(&mt.Header))
			opts = append(opts, grpc.Trailer(&mt.Trailer))
			err = invoker(ctx, method, req, reply, cc, opts...)
			for _, h := range m.afterHandlers {
				h(ctx, header, method, req, reply, cc, err, opts...)
			}
			return err
		}),
	)
}

// Release all rpc client
func (m *Manager) Release() {
	for _, c := range m.addrMap {
		for _, cc := range c {
			for _, ccc := range cc {
				_ = ccc.Close()
			}
		}
	}
}

func (m *Manager) Call(ctx context.Context, from, to, rqId, appid, uid string, cb func(context.Context, *grpc.ClientConn) error) error {
	toM := Module(to)
	addr := m.GetRand(toM)
	if addr == "" {
		return NewRpsError("no rpc addr")
	}
	return m.HostCall(ctx, addr, 1, from, to, rqId, appid, uid, cb)
}

func (m *Manager) ValCall(ctx context.Context, from, to, rqId, appid, uid string, cb func(context.Context, *grpc.ClientConn) (interface{}, error)) (interface{}, error) {
	toM := Module(to)
	addr := m.GetRand(toM)
	if addr == "" {
		return nil, NewRpsError("no rpc addr")
	}
	return m.HostValCall(ctx, addr, 1, from, to, rqId, appid, uid, cb)
}

func (m *Manager) HostCall(ctx context.Context, addr string, flag int, from, to, rqId, appid, uid string, cb func(context.Context, *grpc.ClientConn) error) error {
	cc, err := m.GetConn(Module(to), addr, flag)
	if err != nil {
		return NewRpsError("fetch client failed")
	}

	if cb == nil {
		return NewRpsError("callback is nil")
	}
	ctx1, cl := context.WithTimeout(newRpcMetadataContext(ctx), m.callTtl)
	defer cl()

	ctx1 = metadata.AppendToOutgoingContext(ctx1, "app_id", appid, "user_id", uid, "rq_id", rqId, "rq_type", "rpc", "rq_from", from, "rq_to", to)

	return cb(ctx1, cc)
}

func (m *Manager) HostValCall(ctx context.Context, addr string, flag int, from, to, rqId, appid, uid string, cb func(context.Context, *grpc.ClientConn) (interface{}, error)) (interface{}, error) {
	cc, err := m.GetConn(Module(to), addr, flag)
	if err != nil {
		return nil, NewRpsError("fetch client failed")
	}

	if cb == nil {
		return nil, NewRpsError("callback is nil")
	}
	ctx1, cl := context.WithTimeout(newRpcMetadataContext(ctx), m.callTtl)
	defer cl()

	ctx1 = metadata.AppendToOutgoingContext(ctx1, "app_id", appid, "user_id", uid, "rq_id", rqId, "rq_type", "rpc", "rq_from", from, "rq_to", to)

	return cb(ctx1, cc)
}

func (m *Manager) SetCallTtl(ttl time.Duration) {
	if ttl < 10 {
		ttl = time.Second * ttl
	}
	m.callTtl = ttl
}

func (m *Manager) IsRpsError(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *RpsError
	return errors.As(err, &rpcErr)
}

func (m *Manager) IsCustomError(err error) *CustomError {
	if err == nil {
		return nil
	}
	var rpcErr *CustomError
	if errors.As(err, &rpcErr) {
		return rpcErr
	}
	return nil
}

func (m *Manager) parseHeader(ctx context.Context) Header {
	var rqId, rqFrom, rqTo, appId, userId string
	md, ok := metadata.FromOutgoingContext(ctx)
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

func (m *Manager) SetCustomErrorBuilder(f func(code, message, statusCode string) error) {
	m.errBuilder = f
}
