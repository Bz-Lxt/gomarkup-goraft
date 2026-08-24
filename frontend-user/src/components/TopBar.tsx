import { useApp } from '../store'
import { NODES } from '../types'

export function TopBar() {
  const nodes = useApp((s) => s.nodes)
  const list = NODES.map((id) => nodes[id]).filter(Boolean)
  const leader = list.find((n) => n && n.state === 'leader')
  const term = leader?.term ?? list[0]?.term ?? 0
  const mode = leader?.mode ?? list[0]?.mode ?? 'demo'
  const live = list.filter((n) => n && !n.dead).length

  return (
    <header className="flex w-full items-center justify-between border-b border-grid px-5 py-3">
      <div className="flex items-end gap-4">
        <div>
          <div className="font-display text-xl tracking-wide text-leader">GORAFT</div>
          <div className="text-xs text-mute">声纳指挥甲板 · 强一致配置中心</div>
        </div>
        <span className={`rounded-full border px-3 py-0.5 font-display text-xs uppercase ${mode === 'demo' ? 'border-leader text-leader' : 'border-follower text-follower'}`}>
          {mode} mode
        </span>
      </div>
      <div className="flex gap-6 font-mono text-sm">
        <Metric label="TERM" value={String(term)} />
        <Metric label="LEADER" value={leader?.id ?? '—'} gold />
        <Metric label="LIVE" value={`${live}/5`} />
      </div>
    </header>
  )
}

function Metric({ label, value, gold }: { label: string; value: string; gold?: boolean }) {
  return (
    <div>
      <div className="text-[10px] tracking-[0.2em] text-mute">{label}</div>
      <div className={`text-lg ${gold ? 'text-leader' : 'text-follower'}`}>{value}</div>
    </div>
  )
}
