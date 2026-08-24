import { useApp } from '../store'
import { NODES } from '../types'

export function Inspector() {
  const nodes = useApp((s) => s.nodes)
  return (
    <section className="rounded-xl border border-dashed border-follower/40 bg-panel/80 p-3">
      <h2 className="mb-2 font-display text-sm text-follower">观测通道 · Raft 检视</h2>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        {NODES.map((id) => {
          const n = nodes[id]
          return (
            <div key={id} className="rounded border border-grid p-2 font-mono text-[11px]">
              <div className="font-display text-xs text-leader">{id}</div>
              <div>state {n?.state ?? '—'}</div>
              <div>term {n?.term ?? '—'}</div>
              <div>commit {n?.commit_index ?? '—'}</div>
              <div>applied {n?.last_applied ?? '—'}</div>
              <div>log {n?.log_len ?? '—'}</div>
            </div>
          )
        })}
      </div>
    </section>
  )
}
