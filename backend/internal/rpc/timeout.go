package rpc

import (
	"context"
	"time"
)

func WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		d = 800 * time.Millisecond
	}
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, d)
}
