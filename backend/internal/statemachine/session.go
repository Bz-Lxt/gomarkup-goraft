package statemachine

type Session struct {
	LastSeq uint64
	Last    Result
}

type Sessions struct {
	m map[string]Session
}

func NewSessions() *Sessions {
	return &Sessions{m: map[string]Session{}}
}

func (s *Sessions) Check(clientID string, seq uint64) (hit bool, res Result, skip bool) {
	if clientID == "" || seq == 0 {
		return false, Result{}, false
	}
	cur, ok := s.m[clientID]
	if !ok {
		return false, Result{}, false
	}
	if seq < cur.LastSeq {
		return true, cur.Last, true
	}
	if seq == cur.LastSeq {
		return true, cur.Last, false
	}
	return false, Result{}, false
}

func (s *Sessions) Store(clientID string, seq uint64, res Result) {
	if clientID == "" || seq == 0 {
		return
	}
	s.m[clientID] = Session{LastSeq: seq, Last: res}
}

func (s *Sessions) Snapshot() map[string]Session {
	out := make(map[string]Session, len(s.m))
	for k, v := range s.m {
		out[k] = v
	}
	return out
}

func (s *Sessions) Restore(in map[string]Session) {
	s.m = map[string]Session{}
	for k, v := range in {
		s.m[k] = v
	}
}
