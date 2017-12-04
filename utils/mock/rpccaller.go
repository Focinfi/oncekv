package mock

// RPCClient defines a client to
type RPCClient interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
}

// RPCClientFunc defines Call method of RPCClient interface
type RPCClientFunc func(serviceMethod string, args interface{}, reply interface{}) error

// Call implements RPCClient interface
func (f RPCClientFunc) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return f(serviceMethod, args, reply)
}
