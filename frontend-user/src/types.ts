export type NodeState = 'leader' | 'follower' | 'candidate' | 'learner' | 'unknown'

export interface NodeStatus {
  id: string
  state: NodeState
  term: number
  leader: string
  leader_addr: string
  commit_index: number
  last_applied: number
  last_index: number
  last_term: number
  snap_index: number
  voted_for: string
  voters: string[]
  learners: string[]
  next_index: Record<string, number>
  match_index: Record<string, number>
  mode: string
  dead: boolean
  election_ticks: number
  log_len: number
}

export interface RaftEvent {
  ts: string
  ts_unix_us: number
  node_id: string
  type: string
  term: number
  trace_id?: string
  level?: string
  message?: string
  payload: Record<string, unknown>
}

export interface Span {
  name: string
  node_id: string
  ts_unix_us: number
  ts: string
  detail?: string
  duration_us: number
}

export interface Trace {
  trace_id: string
  key?: string
  op?: string
  spans: Span[]
  done: boolean
}

export const NODES = ['n1', 'n2', 'n3', 'n4', 'n5'] as const
export type NodeId = (typeof NODES)[number]
