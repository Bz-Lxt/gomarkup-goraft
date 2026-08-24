import { useMemo, useState } from 'react'
import { useApp } from '../store'

export function LogStream() {
  const events = useApp((s) => s.events)
  const paused = useApp((s) => s.paused)
  const setPaused = useApp((s) => s.setPaused)
  const [node, setNode] = useState('all')
  const [typ, setTyp] = useState('all')

  const rows = useMemo(() => {
    return events.filter((e) => (node === 'all' || e.node_id === node) && (typ === 'all' || e.type === typ)).slice(-200)
  }, [events, node, typ])

  return (
    <section className="flex h-full min-h-0 flex-col rounded-xl border border-grid bg-panel/80 p-3">
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <h2 className="font-display text-sm">实时日志流</h2>
        <select aria-label="按节点过滤" value={node} onChange={(e) => setNode(e.target.value)} className="rounded border border-grid bg-void px-2 py-1 text-xs">
          <option value="all">全部节点</option>
          {['n1','n2','n3','n4','n5'].map((id) => <option key={id}>{id}</option>)}
        </select>
        <select aria-label="按事件过滤" value={typ} onChange={(e) => setTyp(e.target.value)} className="rounded border border-grid bg-void px-2 py-1 text-xs">
          <option value="all">全部事件</option>
          {['state_change','vote_request','vote_granted','append_send','commit','chaos'].map((t) => <option key={t}>{t}</option>)}
        </select>
        <button onClick={() => setPaused(!paused)} className="rounded border border-leader px-2 py-1 text-xs text-leader">
          {paused ? '继续' : '暂停'}
        </button>
      </div>
      <div className="min-h-0 flex-1 overflow-auto font-mono text-[11px]">
        {rows.map((e, i) => (
          <div key={`${e.ts_unix_us}-${i}`} className="grid grid-cols-[160px_40px_140px_1fr] gap-2 border-b border-grid/60 py-0.5">
            <span className="text-mute">{e.ts}</span>
            <span className="text-leader">{e.node_id}</span>
            <span className="text-follower">{e.type}</span>
            <span>{e.message}</span>
          </div>
        ))}
      </div>
    </section>
  )
}
