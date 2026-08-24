package statemachine

import (
	"encoding/json"
	"strings"
	"sync"

	"goraft/internal/raftpb"
)

type WatchFn func(key, value string, deleted bool)

type KV struct {
	mu      sync.RWMutex
	data    map[string]string
	sess    *Sessions
	watch   []WatchFn
	applied uint64
}

func NewKV() *KV {
	return &KV{data: map[string]string{}, sess: NewSessions()}
}

func (k *KV) Watch(fn WatchFn) {
	k.mu.Lock()
	k.watch = append(k.watch, fn)
	k.mu.Unlock()
}

func (k *KV) Apply(index uint64, typ raftpb.EntryType, data []byte) (any, error) {
	if typ == raftpb.EntryConfig || typ == raftpb.EntryNoop {
		k.mu.Lock()
		k.applied = index
		k.mu.Unlock()
		return nil, nil
	}
	cmd, err := Decode(data)
	if err != nil {
		return nil, err
	}
	k.mu.Lock()
	if hit, res, _ := k.sess.Check(cmd.ClientID, cmd.Seq); hit {
		res.Dup = true
		k.applied = index
		k.mu.Unlock()
		return res, nil
	}
	res := Result{OK: true, Key: cmd.Key}
	deleted := false
	switch cmd.Op {
	case OpPut:
		k.data[cmd.Key] = cmd.Value
		res.Value = cmd.Value
	case OpDelete:
		if _, ok := k.data[cmd.Key]; !ok {
			res.Missing = true
		}
		delete(k.data, cmd.Key)
		deleted = true
	}
	k.sess.Store(cmd.ClientID, cmd.Seq, res)
	k.applied = index
	watchers := append([]WatchFn(nil), k.watch...)
	k.mu.Unlock()
	for _, fn := range watchers {
		fn(cmd.Key, res.Value, deleted)
	}
	return res, nil
}

func (k *KV) fire(key, value string, deleted bool) {
	for _, fn := range k.watch {
		fn(key, value, deleted)
	}
}

func (k *KV) Get(key string) (string, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	v, ok := k.data[key]
	return v, ok
}

func (k *KV) Prefix(prefix string) map[string]string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	out := map[string]string{}
	for key, v := range k.data {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			out[key] = v
		}
	}
	return out
}

func (k *KV) Snapshot() ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	type dump struct {
		Data    map[string]string  `json:"data"`
		Sess    map[string]Session `json:"sess"`
		Applied uint64             `json:"applied"`
	}
	return json.Marshal(dump{Data: k.data, Sess: k.sess.Snapshot(), Applied: k.applied})
}

func (k *KV) Restore(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	type dump struct {
		Data    map[string]string  `json:"data"`
		Sess    map[string]Session `json:"sess"`
		Applied uint64             `json:"applied"`
	}
	var d dump
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}
	if d.Data == nil {
		d.Data = map[string]string{}
	}
	k.mu.Lock()
	k.data = d.Data
	k.sess.Restore(d.Sess)
	k.applied = d.Applied
	k.mu.Unlock()
	return nil
}

func (k *KV) Applied() uint64 {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.applied
}

func (k *KV) Size() int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return len(k.data)
}
