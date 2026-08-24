package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	pool  *Pool
	chaos *Chaos
	from  string
}

func NewClient(pool *Pool, chaos *Chaos, from string) *Client {
	return &Client{pool: pool, chaos: chaos, from: from}
}

func (c *Client) RequestVote(ctx context.Context, addr string, peer string, args RequestVoteArgs) (RequestVoteReply, error) {
	var reply RequestVoteReply
	err := c.call(ctx, addr, peer, "/raft/request-vote", args, &reply)
	return reply, err
}

func (c *Client) AppendEntries(ctx context.Context, addr string, peer string, args AppendEntriesArgs) (AppendEntriesReply, error) {
	var reply AppendEntriesReply
	err := c.call(ctx, addr, peer, "/raft/append-entries", args, &reply)
	return reply, err
}

func (c *Client) InstallSnapshot(ctx context.Context, addr string, peer string, args InstallSnapshotArgs) (InstallSnapshotReply, error) {
	var reply InstallSnapshotReply
	err := c.call(ctx, addr, peer, "/raft/install-snapshot", args, &reply)
	return reply, err
}

func (c *Client) call(ctx context.Context, addr, peer, path string, in any, out any) error {
	if c.chaos != nil {
		drop, delay := c.chaos.Intercept(peer)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		if drop {
			return fmt.Errorf("rpc: chaos drop to %s", peer)
		}
	}
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	url := "http://" + strings.TrimPrefix(addr, "http://") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Raft-From", c.from)
	resp, err := c.pool.Client(peer).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("rpc: status %d from %s", resp.StatusCode, peer)
	}
	return readJSON(resp.Body, maxRPCBody, out)
}
