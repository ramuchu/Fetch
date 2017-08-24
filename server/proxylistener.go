package main

import (
	"errors"
	"fmt"
	"net"
)

type ProxyAddr string

func (pa ProxyAddr) Network() string {
	return string(pa)
}

func (pa ProxyAddr) String() string {
	return string(pa)
}

var defProxyAddr = ProxyAddr("proxynet")

var connClosed = errors.New("Connection closed")

type ProxyListener struct {
	conn    chan net.Conn
	close   chan struct{}
	Address net.Addr
}

func NewProxyListener(addr net.Addr) *ProxyListener {
	return &ProxyListener{
		conn:    make(chan net.Conn),
		close:   make(chan struct{}),
		Address: addr,
	}
}

func (pl *ProxyListener) Conn(conn net.Conn) {
	fmt.Println("Push Conn")
	defer fmt.Println("Pushed Conn")
	pl.conn <- conn
}

func (pl *ProxyListener) Accept() (net.Conn, error) {
	fmt.Println("Accepting...")
	select {
	case c := <-pl.conn:
		fmt.Println("Accepted")
		return c, nil
	case <-pl.close:
		return nil, connClosed
	}
	return nil, connClosed
}

func (pl *ProxyListener) Close() error {
	fmt.Println("Close?")
	close(pl.close)
	return nil
}

func (pl *ProxyListener) Addr() net.Addr {
	if pl.Address != nil {
		return pl.Address
	}
	return defProxyAddr
}
