package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Mode string

const (
	ModeDemo       Mode = "demo"
	ModeProduction Mode = "production"
)

type Config struct {
	ID            string
	HTTPAddr      string
	AdvertiseAddr string
	DataDir       string
	Peers         map[string]string
	Mode          Mode
	LogLevel      string

	TickInterval     time.Duration
	HeartbeatTicks   int
	ElectionTicksMin int
	ElectionTicksMax int

	WALSegmentBytes   int64
	SnapshotLogN      int
	SnapshotBytes     int
	SnapshotChunkSize int

	RPCTimeout     time.Duration
	GroupCommitWin time.Duration
}

func Load() (*Config, error) {
	c := &Config{
		ID:            env("RAFT_ID", "n1"),
		HTTPAddr:      env("RAFT_HTTP_ADDR", ":8080"),
		AdvertiseAddr: env("RAFT_ADV_ADDR", "127.0.0.1:8080"),
		DataDir:       env("RAFT_DATA_DIR", "./data"),
		Mode:          Mode(strings.ToLower(env("RAFT_MODE", "demo"))),
		LogLevel:      env("LOG_LEVEL", "info"),
		WALSegmentBytes: envInt64("RAFT_WAL_SEGMENT", 64<<20),
		SnapshotLogN:    envInt("RAFT_SNAP_LOG_N", 10000),
		SnapshotBytes:   envInt("RAFT_SNAP_BYTES", 8<<20),
		SnapshotChunkSize: envInt("RAFT_SNAP_CHUNK", 64<<10),
		RPCTimeout:      envDur("RAFT_RPC_TIMEOUT", 800*time.Millisecond),
		GroupCommitWin:  envDur("RAFT_GROUP_COMMIT", 2*time.Millisecond),
	}
	peers, err := ParsePeers(env("RAFT_PEERS", c.ID+"="+c.AdvertiseAddr))
	if err != nil {
		return nil, err
	}
	c.Peers = peers
	c.applyModeDefaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) applyModeDefaults() {
	c.TickInterval = 50 * time.Millisecond
	switch c.Mode {
	case ModeProduction:
		c.HeartbeatTicks = 3
		c.ElectionTicksMin = 6
		c.ElectionTicksMax = 12
		if c.TickInterval == 0 {
			c.TickInterval = 25 * time.Millisecond
		}
	default:
		c.Mode = ModeDemo
		c.HeartbeatTicks = 10
		c.ElectionTicksMin = 30
		c.ElectionTicksMax = 60
	}
}

func (c *Config) HeartbeatInterval() time.Duration {
	return c.TickInterval * time.Duration(c.HeartbeatTicks)
}

func (c *Config) ElectionTimeoutMin() time.Duration {
	return c.TickInterval * time.Duration(c.ElectionTicksMin)
}

func (c *Config) ElectionTimeoutMax() time.Duration {
	return c.TickInterval * time.Duration(c.ElectionTicksMax)
}

func ParsePeers(raw string) (map[string]string, error) {
	out := make(map[string]string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out, fmt.Errorf("RAFT_PEERS empty")
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, addr, ok := strings.Cut(part, "=")
		if !ok || strings.TrimSpace(id) == "" || strings.TrimSpace(addr) == "" {
			return nil, fmt.Errorf("invalid peer %q, want id=host:port", part)
		}
		out[strings.TrimSpace(id)] = strings.TrimSpace(addr)
	}
	return out, nil
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func envInt64(k string, def int64) int64 {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return def
}

func envDur(k string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return def
}
