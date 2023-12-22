package rpc

import (
	"context"
	"github.com/obnahsgnaw/application"
	"github.com/obnahsgnaw/application/endtype"
	"github.com/obnahsgnaw/application/pkg/logging/logger"
	"github.com/obnahsgnaw/application/pkg/url"
	"github.com/obnahsgnaw/application/pkg/utils"
	"github.com/obnahsgnaw/application/regtype"
	"github.com/obnahsgnaw/application/servertype"
	"github.com/obnahsgnaw/application/service/regCenter"
	"github.com/obnahsgnaw/http/listener"
	"github.com/obnahsgnaw/rpc/pkg/rpcclient"
	"github.com/obnahsgnaw/rpc/pkg/rpcserver"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"path/filepath"
	"strings"
)

// Server RPC server
type Server struct {
	id         string // 模块
	name       string
	app        *application.Application
	serverType servertype.ServerType
	endType    endtype.EndType
	listener   *listener.PortedListener
	server     *rpcserver.Server
	logger     *zap.Logger
	logCnf     *logger.Config
	manager    *rpcclient.Manager
	services   []ServiceInfo
	regEnable  bool
	regInfos   map[string]*regCenter.RegInfo
	errs       []error
}

type PServer struct {
	Id  string
	Typ servertype.ServerType
}

// ServiceInfo rpc service provider
type ServiceInfo struct {
	Desc grpc.ServiceDesc
	Impl interface{}
}

func New(app *application.Application, id, name string, et endtype.EndType, lr *listener.PortedListener, ps *PServer, options ...Option) *Server {
	var err error
	s := &Server{
		id:         id,
		name:       name,
		app:        app,
		serverType: servertype.Rpc,
		endType:    et,
		listener:   lr,
		manager:    rpcclient.NewManager(),
		regInfos:   make(map[string]*regCenter.RegInfo),
	}
	if s.id == "" || s.name == "" {
		s.addErr(s.err("id or name invalid", nil))
	}
	s.logCnf = logger.CopyCnfWithLevel(s.app.LogConfig())
	if s.logCnf != nil {
		if ps != nil {
			s.logCnf.AddSubDir(filepath.Join(s.endType.String(), utils.ToStr(ps.Typ.String(), "-", ps.Id), utils.ToStr(s.serverType.String(), "-", s.id)))
		} else {
			s.logCnf.AddSubDir(filepath.Join(s.endType.String(), utils.ToStr(s.serverType.String(), "-", s.id)))
		}
		s.logCnf.ReplaceTraceLevel(zap.NewAtomicLevelAt(zap.FatalLevel))
		s.logCnf.SetFilename(utils.ToStr(s.serverType.String(), "-", s.id))
	}
	s.logger, err = logger.New(utils.ToStr(s.serverType.String(), ":", s.endType.String(), "-", s.id), s.logCnf, s.app.Debugger().Debug())
	s.addErr(err)
	s.With(options...)
	s.server = rpcserver.New(s.listener, s.logger)
	s.manager.RegisterAfterHandler(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, err error, opts ...grpc.CallOption) {
		if err != nil {
			s.logger.Error(utils.ToStr("rpc call[", method, "] failed, ", err.Error()), zap.Any("req", req), zap.Any("resp", reply))
		} else {
			s.logger.Debug(utils.ToStr("rpc call[", method, "] success"), zap.Any("req", req), zap.Any("resp", reply))
		}
	})
	s.AddRegInfo(id, name, ps)
	return s
}

func Default(app *application.Application, id, name string, et endtype.EndType, host url.Host, ps *PServer, options ...Option) (*Server, error) {
	if l, err := listener.Default(host); err != nil {
		return nil, err
	} else {
		return New(app, id, name, et, l, ps, options...), nil
	}
}

func (s *Server) With(options ...Option) {
	for _, o := range options {
		o(s)
	}
}

// ID return the server id
func (s *Server) ID() string {
	return s.id
}

// Name return the server name
func (s *Server) Name() string {
	return s.name
}

// Type return the server type
func (s *Server) Type() servertype.ServerType {
	return s.serverType
}

// EndType return the server end type
func (s *Server) EndType() endtype.EndType {
	return s.endType
}

// Release resource
func (s *Server) Release() {
	if s.RegEnabled() && s.app.Register() != nil {
		for _, info := range s.regInfos {
			_ = s.app.DoUnregister(info, func(msg string) {
				if s.logger != nil {
					s.logger.Debug(msg)
				}
			})
		}
	}
	s.manager.Release()
	if s.logger != nil {
		s.logger.Info("released")
		_ = s.logger.Sync()
	}
}

// Run server
func (s *Server) Run(failedCb func(error)) {
	if len(s.errs) > 0 {
		failedCb(s.errs[0])
		return
	}
	s.logger.Info("init starting...")
	s.server.RegisterAfterHandler(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, resp interface{}, err error) {
		if err != nil {
			s.logger.Error(utils.ToStr("rpc serve[", info.FullMethod, "] failed, ", err.Error()), zap.Any("req", req), zap.Any("resp", resp))
		} else {
			s.logger.Debug(utils.ToStr("rpc serve[", info.FullMethod, "] success"), zap.Any("req", req), zap.Any("resp", resp))
		}
	})
	for _, sp := range s.services {
		s.server.Register(&sp.Desc, sp.Impl)
		s.logger.Debug(utils.ToStr("service[", sp.Desc.ServiceName, "] registered"))
	}
	if len(s.services) == 0 {
		s.logger.Warn("no service registered")
	}
	s.logger.Info("services initialized")
	if s.app.Register() != nil {
		if s.RegEnabled() {
			s.logger.Debug("server register start...")
			for id, info := range s.regInfos {
				if err := s.app.DoRegister(info, func(msg string) {
					s.logger.Debug(msg)
				}); err != nil {
					failedCb(s.err("register failed", err))
					return
				}
				s.logger.Debug("server[" + id + "] registered")
			}
			s.logger.Debug("server register initialized")
		}
		s.logger.Debug("server watch started")
		if err := s.watch(s.app.Register()); err != nil {
			failedCb(s.err("watch failed", err))
			return
		}
	}
	s.logger.Info("register initialized")
	s.logger.Info("initialized")
	s.server.SyncStart(func(err error) {
		failedCb(s.err("run failed, err="+err.Error(), nil))
	})
	s.logger.Info(utils.ToStr("server[", s.Host().String(), "] listen and serving..."))
}

// Host return the server host
func (s *Server) Host() url.Host {
	return s.listener.Host()
}

func (s *Server) Listener() *listener.PortedListener {
	return s.listener
}

// AddRegInfo 添加注册信息，多个服务用一个rpc时， 当然得同一个 endtype
func (s *Server) AddRegInfo(id, name string, parent *PServer) {
	st := s.serverType.String()
	if parent != nil {
		st = parent.Typ.String()
	}
	s.regInfos[id] = &regCenter.RegInfo{
		AppId:   s.app.ID(),
		RegType: regtype.Rpc,
		ServerInfo: regCenter.ServerInfo{
			Id:      id,
			Name:    name,
			Type:    st,
			EndType: s.endType.String(),
		},
		Host:      s.listener.Host().String(),
		Val:       s.listener.Host().String(),
		Ttl:       s.app.RegTtl(),
		KeyPreGen: regCenter.DefaultRegKeyPrefixGenerator(),
	}
}

// RegEnabled reg enabled
func (s *Server) RegEnabled() bool {
	return s.regEnable
}

// RegInfo return the server register info
func (s *Server) RegInfo() map[string]*regCenter.RegInfo {
	return s.regInfos
}

// Manager return rpc manager
func (s *Server) Manager() *rpcclient.Manager {
	return s.manager
}

// Logger return the logger
func (s *Server) Logger() *zap.Logger {
	return s.logger
}

// LogConfig return
func (s *Server) LogConfig() *logger.Config {
	return s.logCnf
}

// RegisterService register a rcp service
func (s *Server) RegisterService(provider ServiceInfo) {
	s.services = append(s.services, provider)
}

func (s *Server) watch(register regCenter.Register) (err error) {
	// watch rpc
	if s.regEnable {
		prefix := s.regInfos[s.id].Prefix()
		if prefix == "" {
			return s.err("reg key prefix is empty", nil)
		}
		return register.Watch(s.app.Context(), prefix, func(key string, val string, isDel bool) {
			segments := strings.Split(key, "/")
			module := segments[len(segments)-2]
			addr := segments[len(segments)-1]
			if isDel {
				s.logger.Debug(utils.ToStr("rpc[", module, "] leaved"))
				s.manager.Rm(rpcclient.Module(module), addr)
			} else {
				s.logger.Debug(utils.ToStr("rpc[", module, "] added"))
				s.manager.Add(rpcclient.Module(module), addr)
			}
		})
	}
	return
}

func (s *Server) err(msg string, err error) error {
	return utils.TitledError(utils.ToStr("rpc server[", s.name, "] error"), msg, err)
}

func (s *Server) addErr(err error) {
	if err != nil {
		s.errs = append(s.errs, err)
	}
}
