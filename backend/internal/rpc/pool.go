package rpc

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type Pool struct {
	mu        sync.Mutex
	clients   map[string]*http.Client
	timeout   time.Duration
	transport *http.Transport
}

func NewPool(timeout time.Duration) *Pool {
	if timeout <= 0 {
		timeout = 800 * time.Millisecond
	}
	tr := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   400 * time.Millisecond,
			KeepAlive: 15 * time.Second,
		}).DialContext,
		MaxIdleConns:        64,
		MaxIdleConnsPerHost: 8,
		IdleConnTimeout:     8 * time.Second,
		DisableCompression:  true,
	}
	return &Pool{
		clients:   map[string]*http.Client{},
		timeout:   timeout,
		transport: tr,
	}
}

func (p *Pool) Client(_ string) *http.Client {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := "shared"
	if c, ok := p.clients[key]; ok {
		return c
	}
	c := &http.Client{Transport: p.transport, Timeout: p.timeout}
	p.clients[key] = c
	return c
}

func (p *Pool) Close() {
	p.transport.CloseIdleConnections()
}
