package main

import (
	"github.com/obnahsgnaw/application"
	"github.com/obnahsgnaw/application/endtype"
	"github.com/obnahsgnaw/application/pkg/url"
	"github.com/obnahsgnaw/rpc"
)

func main() {
	app := application.New(application.NewCluster("dev", "Dev"), "RpcDemo")
	defer app.Release()

	app.With(application.Debug(func() bool {
		return true
	}))

	s := rpc.New(app, "auth", "auth", endtype.Backend, url.Host{Ip: "127.0.0.1", Port: 7001})

	app.AddServer(s)

	app.Run(func(err error) {
		panic(err)
	})

	app.Wait()
}
