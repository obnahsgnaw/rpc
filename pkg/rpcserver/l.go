package rpcserver

import "net"

type NoCloseListener struct {
	l net.Listener
}

func newNoCl(l net.Listener) *NoCloseListener {
	return &NoCloseListener{l: l}
}

// Accept waits for and returns the next connection to the listener.
func (s *NoCloseListener) Accept() (net.Conn, error) {
	return s.l.Accept()
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (s *NoCloseListener) Close() error {
	return nil
}

// Addr returns the listener's network address.
func (s *NoCloseListener) Addr() net.Addr {
	return s.l.Addr()
}
