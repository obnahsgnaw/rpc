package rpc

import (
	"github.com/obnahsgnaw/application"
	"github.com/obnahsgnaw/application/endtype"
	"github.com/obnahsgnaw/application/pkg/logging/logger"
	"github.com/obnahsgnaw/application/pkg/url"
	"github.com/obnahsgnaw/application/pkg/utils"
	"github.com/obnahsgnaw/application/regtype"
	"github.com/obnahsgnaw/application/servertype"
	"github.com/obnahsgnaw/application/service/regCenter"
	"github.com/obnahsgnaw/rpc/pkg/rpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"strings"
)

// Server RPC server
type Server struct {
	id        string // 模块
	name      string
	st        servertype.ServerType
	et        endtype.EndType
	host      url.Host
	server    *rpc.Server
	services  []ServiceInfo
	app       *application.Application
	regEnable bool
	regInfo   *regCenter.RegInfo
	pServer   application.Server // 依附的上级服务 如api 或 tcp...
	logger    *zap.Logger
	manager   *rpc.Manager
	errs      []error
}

// ServiceInfo rpc service provider
type ServiceInfo struct {
	Desc grpc.ServiceDesc
	Impl interface{}
}

func New(app *application.Application, id, name string, et endtype.EndType, host url.Host, options ...Option) *Server {
	var err error
	s := &Server{
		id:      id,
		name:    name,
		st:      servertype.Rpc,
		et:      et,
		host:    host,
		app:     app,
		manager: rpc.NewManager(),
	}
	if s.id == "" || s.name == "" {
		s.addErr(s.err("id or name invalid", nil))
	}

	if s.host.Port == 0 || s.host.Ip == "" {
		s.addErr(s.err("host invalid", nil))
	}
	s.logger, err = logger.New(utils.ToStr("Rpc[", s.et.String(), "][", id, "]"), s.app.LogConfig(), s.app.Debugger().Debug())
	s.addErr(err)

	s.regInfo = &regCenter.RegInfo{
		AppId:   s.app.ID(),
		RegType: regtype.Rpc,
		ServerInfo: regCenter.ServerInfo{
			Id:      s.id,
			Name:    s.name,
			Type:    s.st.String(),
			EndType: s.et.String(),
		},
		Host:      s.host.String(),
		Val:       s.host.String(),
		Ttl:       s.app.RegTtl(),
		KeyPreGen: regCenter.DefaultRegKeyPrefixGenerator(),
	}
	s.With(options...)
	return s
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
	return s.st
}

// EndType return the server end type
func (s *Server) EndType() endtype.EndType {
	return s.et
}

// Host return the server host
func (s *Server) Host() url.Host {
	return s.host
}

// RegEnabled reg enabled
func (s *Server) RegEnabled() bool {
	return s.regEnable
}

// RegInfo return the server register info
func (s *Server) RegInfo() *regCenter.RegInfo {
	return s.regInfo
}

// Manager return rpc manager
func (s *Server) Manager() *rpc.Manager {
	return s.manager
}

// Logger return the logger
func (s *Server) Logger() *zap.Logger {
	return s.logger
}

// RegisterService register a rcp service
func (s *Server) RegisterService(provider ServiceInfo) {
	s.services = append(s.services, provider)
}

// Release resource
func (s *Server) Release() {
	if s.RegEnabled() {
		_ = s.app.DoUnregister(s.regInfo)
	}
	s.manager.Release()
	_ = s.logger.Sync()
	s.debug("released")
}

// Run server
func (s *Server) Run(failedCb func(error)) {
	if len(s.errs) > 0 {
		failedCb(s.errs[0])
		return
	}
	s.logger.Info("start running...")
	ss, err := rpc.NewServer(s.host.Port)
	if err != nil {
		failedCb(s.err("new server failed", err))
		return
	}
	s.server = ss
	for _, sp := range s.services {
		s.server.Register(&sp.Desc, sp.Impl)
		s.debug(utils.ToStr("rpc service[", sp.Desc.ServiceName, "] registered"))
	}

	if s.app.Register() != nil {
		if s.RegEnabled() {
			if err = s.app.DoRegister(s.regInfo); err != nil {
				failedCb(s.err("register failed", err))
				return
			}
		}
		if err = s.watch(s.app.Register()); err != nil {
			failedCb(s.err("watch failed", err))
			return
		}
	}

	s.server.SyncStart(func(err error) {
		failedCb(s.err("run failed, err="+err.Error(), nil))
	})
	s.logger.Info(utils.ToStr("rpc server[", s.host.String(), "] listen and serving..."))
}

func (s *Server) watch(register regCenter.Register) (err error) {
	// watch rpc
	if s.regEnable {
		prefix := s.regInfo.Prefix()
		if prefix == "" {
			return s.err("reg key prefix is empty", nil)
		}
		return register.Watch(s.app.Context(), prefix, func(key string, val string, isDel bool) {
			segments := strings.Split(key, "/")
			module := segments[len(segments)-2]
			addr := segments[len(segments)-1]
			if isDel {
				s.debug(utils.ToStr("rpc[", module, "] leaved"))
				s.manager.Rm(rpc.Module(module), addr)
			} else {
				s.debug(utils.ToStr("rpc[", module, "] added"))
				s.manager.Add(rpc.Module(module), addr)
			}
		})
	}
	return
}

func (s *Server) err(msg string, err error) error {
	return utils.TitledError(utils.ToStr("rpc server[", s.name, "] error"), msg, err)
}

func (s *Server) debug(msg string) {
	if s.app.Debugger().Debug() {
		s.logger.Debug(msg)
	}
}

func (s *Server) addErr(err error) {
	if err != nil {
		s.errs = append(s.errs, err)
	}
}
