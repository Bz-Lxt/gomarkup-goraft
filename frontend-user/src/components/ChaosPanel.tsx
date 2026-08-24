import { useState } from 'react'
import { chaos } from '../lib/api'
import { useApp } from '../store'
import { NODES, type NodeId } from '../types'

export function ChaosPanel() {
  const ask = useApp((s) => s.ask)
  const toast = useApp((s) => s.toast)
  const [target, setTarget] = useState<NodeId>('n1')
  const [peer, setPeer] = useState<NodeId>('n2')

  const run = (title: string, body: string, fn: () => Promise<void>) => ask(title, body, fn)

  return (
    <section className="rounded-xl border border-danger/40 bg-panel/80 p-3">
      <h2 className="mb-2 font-display text-sm text-danger">Chaos 控制台</h2>
      <div className="mb-2 flex gap-2">
        <select aria-label="目标节点" value={target} onChange={(e) => setTarget(e.target.value as NodeId)} className="rounded border border-grid bg-void px-2 py-1 text-xs">
          {NODES.map((id) => <option key={id}>{id}</option>)}
        </select>
        <select aria-label="对端节点" value={peer} onChange={(e) => setPeer(e.target.value as NodeId)} className="rounded border border-grid bg-void px-2 py-1 text-xs">
          {NODES.map((id) => <option key={id}>{id}</option>)}
        </select>
      </div>
      <div className="flex flex-wrap gap-2">
        <Btn label="Kill Leader 目标" onClick={() => run('确认 Kill', `停止 ${target} 的 tick 与 RPC，模拟断电。`, async () => { await chaos(target, 'kill'); toast(`${target} 已 Kill`) })} />
        <Btn label="Revive" onClick={() => run('确认 Revive', `恢复 ${target}`, async () => { await chaos(target, 'revive'); toast(`${target} 已恢复`) })} />
        <Btn label="分区隔离" onClick={() => run('确认分区', `${target} ↔ ${peer} 双向丢包`, async () => { await chaos(target, 'partition', { peer }); await chaos(peer, 'partition', { peer: target }); toast('分区已注入') })} />
        <Btn label="愈合" onClick={() => run('确认愈合', '清除全部分区/延迟规则', async () => { await Promise.all(NODES.map((id) => chaos(id, 'heal'))); toast('集群已愈合') })} />
        <Btn label="注入延迟" onClick={() => run('确认延迟', `${target} → ${peer} 延迟 400ms`, async () => { await chaos(target, 'delay', { peer, delay_ms: 400 }); toast('延迟已注入') })} />
      </div>
    </section>
  )
}

function Btn({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button onClick={onClick} className="rounded border border-danger/60 px-2 py-1 text-xs text-danger hover:bg-danger/10">
      {label}
    </button>
  )
}
