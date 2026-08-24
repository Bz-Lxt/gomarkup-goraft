import type { NodeId, NodeStatus, RaftEvent, Trace } from '../types'
import { NODES } from '../types'

async function req(node: NodeId, path: string, init?: RequestInit) {
  const res = await fetch(`/${node}${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
  })
  const text = await res.text()
  let body: any = null
  try { body = text ? JSON.parse(text) : null } catch { body = { error: text } }
  if (!res.ok) {
    const err = new Error(body?.error || res.statusText)
    ;(err as any).body = body
    ;(err as any).status = res.status
    throw err
  }
  return body
}

export async function fetchState(node: NodeId): Promise<NodeStatus | null> {
  try {
    const body = await req(node, '/api/v1/observe/state')
    return body.status as NodeStatus
  } catch {
    return null
  }
}

export async function fetchAllStates(): Promise<Record<string, NodeStatus | null>> {
  const entries = await Promise.all(NODES.map(async (id) => [id, await fetchState(id)] as const))
  return Object.fromEntries(entries)
}

export async function putKV(node: NodeId, key: string, value: string) {
  return req(node, `/api/v1/kv/${encodeURIComponent(key)}`, {
    method: 'PUT',
    body: JSON.stringify({ value, client_id: 'dashboard', seq: Date.now() }),
  })
}

export async function getKV(node: NodeId, key: string, stale = false) {
  return req(node, `/api/v1/kv/${encodeURIComponent(key)}${stale ? '?stale=true' : ''}`)
}

export async function chaos(node: NodeId, action: string, body: Record<string, unknown> = {}) {
  return req(node, `/api/v1/chaos/${action}`, { method: 'POST', body: JSON.stringify(body) })
}

export async function fetchTraces(node: NodeId): Promise<Trace[]> {
  try {
    const body = await req(node, '/api/v1/observe/traces')
    return body.traces || []
  } catch {
    return []
  }
}

export async function fetchLogs(node: NodeId, after = 0): Promise<RaftEvent[]> {
  try {
    const body = await req(node, `/api/v1/observe/logs?after=${after}&limit=200`)
    return body.events || []
  } catch {
    return []
  }
}

export function openWS(node: NodeId, onEvent: (ev: RaftEvent) => void): WebSocket {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const ws = new WebSocket(`${proto}://${location.host}/${node}/api/v1/ws`)
  ws.onmessage = (m) => {
    try { onEvent(JSON.parse(m.data)) } catch { /* ignore */ }
  }
  return ws
}
