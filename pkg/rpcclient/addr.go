package rpcclient

import "google.golang.org/grpc"

type Addr map[string]map[int]*grpc.ClientConn

func (a Addr) Add(server string) {
	a[server] = make(map[int]*grpc.ClientConn)
}
