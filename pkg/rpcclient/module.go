package rpcclient

type Module string

func (m Module) String() string {
	return string(m)
}
