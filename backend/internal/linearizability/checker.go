package linearizability

import "fmt"

type call struct {
	ev   Event
	ok   *Event
	fail bool
}

// Check verifies a sequential specification of a multi-register KV
// against a concurrent history (simplified Jepsen/Knossos style).
func Check(h History) error {
	invokes := map[int]Event{}
	var ops []call
	for _, e := range h {
		switch e.Kind {
		case Invoke:
			invokes[e.ID] = e
		case OK:
			inv, ok := invokes[e.ID]
			if !ok {
				return fmt.Errorf("ok without invoke id=%d", e.ID)
			}
			ops = append(ops, call{ev: inv, ok: &e})
			delete(invokes, e.ID)
		case Fail:
			inv, ok := invokes[e.ID]
			if !ok {
				return fmt.Errorf("fail without invoke id=%d", e.ID)
			}
			ops = append(ops, call{ev: inv, fail: true})
			delete(invokes, e.ID)
		}
	}
	for _, inv := range invokes {
		ops = append(ops, call{ev: inv, fail: true})
	}
	state := map[string]string{}
	return search(ops, 0, state, make([]bool, len(ops)))
}

func search(ops []call, linearized int, state map[string]string, used []bool) error {
	if linearized == len(ops) {
		return nil
	}
	for i, op := range ops {
		if used[i] {
			continue
		}
		if !canLinearize(ops, i, used) {
			continue
		}
		next, ok := step(state, op)
		if !ok {
			continue
		}
		used[i] = true
		if err := search(ops, linearized+1, next, used); err == nil {
			return nil
		}
		used[i] = false
	}
	return fmt.Errorf("history is not linearizable")
}

func canLinearize(ops []call, i int, used []bool) bool {
	a := ops[i]
	for j, b := range ops {
		if used[j] || j == i {
			continue
		}
		if !b.fail && b.ok != nil && b.ok.Time.Before(a.ev.Time) {
			return false
		}
	}
	return true
}

func step(state map[string]string, op call) (map[string]string, bool) {
	if op.fail {
		return clone(state), true
	}
	next := clone(state)
	switch op.ev.Op {
	case OpPut:
		next[op.ev.Key] = op.ev.Value
		return next, true
	case OpDelete:
		delete(next, op.ev.Key)
		return next, true
	case OpGet:
		got := ""
		if op.ok != nil {
			got = op.ok.Value
		}
		cur, ok := next[op.ev.Key]
		if !ok {
			return next, got == ""
		}
		return next, cur == got
	default:
		return nil, false
	}
}

func clone(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
