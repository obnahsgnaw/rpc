package rpc

import (
	"github.com/obnahsgnaw/application/pkg/url"
	"github.com/obnahsgnaw/http/listener"
)

func NewListener(host url.Host) (*listener.PortedListener, error) {
	return listener.Default(host)
}
