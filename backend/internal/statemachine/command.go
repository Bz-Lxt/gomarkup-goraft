package statemachine

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	OpPut    = "put"
	OpDelete = "delete"
)

type Command struct {
	Op       string `json:"op"`
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	ClientID string `json:"client_id"`
	Seq      uint64 `json:"seq"`
}

func (c Command) Validate() error {
	c.Op = strings.ToLower(strings.TrimSpace(c.Op))
	if c.Op != OpPut && c.Op != OpDelete {
		return fmt.Errorf("unsupported op %q", c.Op)
	}
	if strings.TrimSpace(c.Key) == "" {
		return fmt.Errorf("key required")
	}
	if len(c.Key) > 256 {
		return fmt.Errorf("key too long")
	}
	if len(c.Value) > 1<<20 {
		return fmt.Errorf("value too large")
	}
	return nil
}

func Encode(c Command) ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c)
}

func Decode(b []byte) (Command, error) {
	var c Command
	if len(b) == 0 {
		return c, fmt.Errorf("empty command")
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("command json: %w", err)
	}
	return c, c.Validate()
}

type Result struct {
	OK      bool   `json:"ok"`
	Key     string `json:"key"`
	Value   string `json:"value,omitempty"`
	Missing bool   `json:"missing,omitempty"`
	Dup     bool   `json:"dup,omitempty"`
}
