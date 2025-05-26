package connContainer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type Container struct {
	serverConn       Conn
	clientConn       Conn
	hasSetTunnelChan chan struct{}
	hasSetTunnel     atomic.Bool
	tunnelRWMutex    sync.RWMutex
	typ              uint8
}

func NewServerConn(conn Conn) *Container {
	c := &Container{
		serverConn:       conn,
		hasSetTunnelChan: make(chan struct{}),
		typ:              1,
	}

	return c
}

func NewClientConn(conn Conn) *Container {
	c := &Container{
		clientConn:       conn,
		hasSetTunnelChan: make(chan struct{}),
		typ:              2,
	}

	return c
}

func (c *Container) SetServerConn(conn Conn) bool {
	if c.typ == 2 {
		if c.hasSetTunnel.CompareAndSwap(false, true) {
			c.serverConn = conn
			close(c.hasSetTunnelChan)
			return true
		}
	}

	return false
}

func (c *Container) SetClientConn(conn Conn) bool {
	if c.typ == 1 {
		if c.hasSetTunnel.CompareAndSwap(false, true) {
			c.clientConn = conn
			close(c.hasSetTunnelChan)
			return true
		}
	}

	return false
}

func (c *Container) WaitSetTunnel(ctx context.Context) (clientConn Conn, serverConn Conn, err error) {
	select {
	case <-c.hasSetTunnelChan:
		return c.clientConn, c.serverConn, nil
	case <-ctx.Done():
		if c.hasSetTunnel.CompareAndSwap(false, true) {
			switch c.typ {
			case 1:
				return nil, nil, fmt.Errorf("等待客户端隧道超时")
			case 2:
				return nil, nil, fmt.Errorf("等待服务端隧道超时")
			}
		}
	}
	<-c.hasSetTunnelChan
	return c.clientConn, c.serverConn, nil
}
