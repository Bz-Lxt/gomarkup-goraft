package config

import (
	"fmt"
	"strings"
)

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("RAFT_ID required")
	}
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return fmt.Errorf("RAFT_HTTP_ADDR required")
	}
	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("RAFT_DATA_DIR required")
	}
	if len(c.Peers) == 0 {
		return fmt.Errorf("RAFT_PEERS required")
	}
	if _, ok := c.Peers[c.ID]; !ok {
		return fmt.Errorf("RAFT_PEERS must include self id %s", c.ID)
	}
	if c.TickInterval <= 0 {
		return fmt.Errorf("tick interval must be > 0")
	}
	if c.HeartbeatTicks <= 0 || c.ElectionTicksMin <= c.HeartbeatTicks {
		return fmt.Errorf("election ticks must exceed heartbeat ticks")
	}
	if c.ElectionTicksMax < c.ElectionTicksMin {
		return fmt.Errorf("election ticks max < min")
	}
	if c.WALSegmentBytes < 1<<20 {
		return fmt.Errorf("WAL segment must be >= 1MiB")
	}
	if c.SnapshotChunkSize < 1024 {
		return fmt.Errorf("snapshot chunk must be >= 1KiB")
	}
	return nil
}

func (c *Config) PeerIDs() []string {
	ids := make([]string, 0, len(c.Peers))
	for id := range c.Peers {
		ids = append(ids, id)
	}
	return ids
}
