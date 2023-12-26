package rpcclient

import "google.golang.org/grpc"

type Addr map[string]map[int]*grpc.ClientConn

func (a Addr) Add(server string) {
	if _, ok := a[server]; !ok {
		a[server] = make(map[int]*grpc.ClientConn)
	}
}
