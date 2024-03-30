package main

import (
	"github.com/obnahsgnaw/application"
	"github.com/obnahsgnaw/application/endtype"
	"github.com/obnahsgnaw/application/pkg/logging/logger"
	"github.com/obnahsgnaw/application/pkg/url"
	"github.com/obnahsgnaw/application/service/regCenter"
	"github.com/obnahsgnaw/rpc"
	"time"
)

func main() {
	r, _ := regCenter.NewEtcdRegister([]string{"127.0.0.1:2379"}, time.Second*5)
	app := application.New(
		"demo",
		application.Debug(func() bool {
			return true
		}),
		application.Logger(&logger.Config{
			Dir:        "/Users/wangshanbo/Documents/Data/projects/rpc/out",
			MaxSize:    5,
			MaxBackup:  1,
			MaxAge:     1,
			Level:      "debug",
			TraceLevel: "error",
		}),
		application.Register(r, 5),
	)
	defer app.Release()

	l, _ := rpc.NewListener(url.Host{Ip: "127.0.0.1", Port: 7001})

	s := rpc.New(
		app,
		l,
		"auth",
		"auth",
		endtype.Backend,
		nil, // rpc.NewPServer("", "")
		//rpc.RegEnable(),
		//rpc.IgLrClose(),
	)
	app.AddServer(s)
	app.Run(func(err error) {
		panic(err)
	})
	app.Wait()
}
