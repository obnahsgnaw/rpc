package main

import (
	"github.com/obnahsgnaw/application"
	"github.com/obnahsgnaw/application/endtype"
	"github.com/obnahsgnaw/application/pkg/logging/logger"
	"github.com/obnahsgnaw/application/pkg/url"
	"github.com/obnahsgnaw/rpc"
	"time"
)

func main() {
	app := application.New(application.NewCluster("dev", "Dev"), "demo")
	defer app.Release()

	app.With(application.Debug(func() bool {
		return true
	}))
	app.With(application.Logger(&logger.Config{
		Dir:        "/Users/wangshanbo/Documents/Data/projects/rpc/out",
		MaxSize:    5,
		MaxBackup:  1,
		MaxAge:     1,
		Level:      "debug",
		TraceLevel: "error",
	}))
	app.With(application.EtcdRegister([]string{"127.0.0.1:2379"}, time.Second*5))

	s := rpc.New(app, "auth", "auth", endtype.Backend, url.Host{Ip: "127.0.0.1", Port: 7001})

	app.AddServer(s)

	app.Run(func(err error) {
		panic(err)
	})

	app.Wait()
}
