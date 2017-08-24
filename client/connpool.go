package main

import "net"

// ConnPool is an interface for create / retrieve conn
type ConnPool interface {
	Get() (net.Conn, error)
	Put(net.Conn)
}

// SimplePool type is an adapter to allow the use of ordinary functions as ConnPool
type SimplePool func() (net.Conn, error)

// Get calls p().
func (p SimplePool) Get() (net.Conn, error) {
	//fmt.Println("  >> GET")
	return p()
}

// Put will call conn.Close().
func (p SimplePool) Put(conn net.Conn) {
	//fmt.Println("<<   Put")
	conn.Close()
}
