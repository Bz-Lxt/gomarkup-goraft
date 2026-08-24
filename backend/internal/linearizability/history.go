package linearizability

import "time"

type Kind uint8

const (
	Invoke Kind = iota
	OK
	Fail
)

type OpType string

const (
	OpPut    OpType = "put"
	OpGet    OpType = "get"
	OpDelete OpType = "delete"
)

type Event struct {
	Kind  Kind
	ID    int
	Proc  int
	Op    OpType
	Key   string
	Value string
	Time  time.Time
}

type History []Event
