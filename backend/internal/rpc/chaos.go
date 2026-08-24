package rpc

import (
	"math/rand"
	"sync"
	"time"
)

type Rule struct {
	DropRate float64       `json:"drop_rate"`
	Delay    time.Duration `json:"delay"`
	Blocked  bool          `json:"blocked"`
}

type Chaos struct {
	mu     sync.RWMutex
	dead   bool
	global Rule
	peers  map[string]Rule
}

func NewChaos() *Chaos {
	return &Chaos{peers: map[string]Rule{}}
}

func (c *Chaos) Kill() {
	c.mu.Lock()
	c.dead = true
	c.mu.Unlock()
}

func (c *Chaos) Revive() {
	c.mu.Lock()
	c.dead = false
	c.mu.Unlock()
}

func (c *Chaos) Dead() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dead
}

func (c *Chaos) SetPeer(id string, r Rule) {
	c.mu.Lock()
	if r.DropRate < 0 {
		r.DropRate = 0
	}
	if r.DropRate > 1 {
		r.DropRate = 1
	}
	if r.Delay < 0 {
		r.Delay = 0
	}
	c.peers[id] = r
	c.mu.Unlock()
}

func (c *Chaos) ClearPeer(id string) {
	c.mu.Lock()
	delete(c.peers, id)
	c.mu.Unlock()
}

func (c *Chaos) Heal() {
	c.mu.Lock()
	c.peers = map[string]Rule{}
	c.global = Rule{}
	c.mu.Unlock()
}

func (c *Chaos) Snapshot() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	peers := map[string]Rule{}
	for k, v := range c.peers {
		peers[k] = v
	}
	return map[string]any{"dead": c.dead, "peers": peers}
}

func (c *Chaos) Intercept(peer string) (drop bool, delay time.Duration) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.dead {
		return true, 0
	}
	r := c.peers[peer]
	if r.Blocked {
		return true, r.Delay
	}
	if r.DropRate > 0 && rand.Float64() < r.DropRate {
		return true, r.Delay
	}
	return false, r.Delay
}
