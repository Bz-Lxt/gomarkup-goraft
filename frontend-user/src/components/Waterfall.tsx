import { useApp } from '../store'

export function Waterfall() {
  const traces = useApp((s) => s.traces)
  const selected = useApp((s) => s.selectedTrace)
  const setSelected = useApp((s) => s.setSelectedTrace)
  const current = traces.find((t) => t.trace_id === selected) || traces[traces.length - 1]

  return (
    <section className="flex h-full min-h-0 flex-col rounded-xl border border-leader/40 bg-panel/80 p-3">
      <div className="mb-2 flex items-center justify-between">
        <h2 className="font-display text-sm text-leader">复制时序瀑布流</h2>
        <span className="text-[10px] text-mute">业务通道 · 真实微秒</span>
      </div>
      <div className="mb-2 flex gap-2 overflow-x-auto">
        {traces.slice(-8).map((t) => (
          <button
            key={t.trace_id}
            aria-label={`trace ${t.trace_id}`}
            onClick={() => setSelected(t.trace_id)}
            className={`whitespace-nowrap rounded border px-2 py-1 font-mono text-[10px] ${selected === t.trace_id ? 'border-leader text-leader' : 'border-grid text-mute'}`}
          >
            {t.op || 'write'} {t.key}
          </button>
        ))}
      </div>
      <div className="min-h-0 flex-1 overflow-auto">
        {!current && <div className="text-sm text-mute">发起一次写入后，这里会展开 WAL → 复制 → 多数派 → Commit。</div>}
        {current?.spans.map((sp, i) => {
          const max = Math.max(...current.spans.map((s) => s.duration_us || 1), 1)
          const w = Math.max(8, (sp.duration_us / max) * 100)
          return (
            <div key={`${sp.ts_unix_us}-${i}`} className="mb-2">
              <div className="flex justify-between font-mono text-[11px]">
                <span>{sp.name} · {sp.node_id}</span>
                <span className="text-follower">{sp.duration_us}μs</span>
              </div>
              <div className="h-2 rounded bg-void">
                <div className="h-2 rounded bg-follower" style={{ width: `${w}%` }} />
              </div>
              <div className="text-[10px] text-mute">{sp.ts}</div>
            </div>
          )
        })}
      </div>
    </section>
  )
}
