import { useState } from 'react'
import { getKV, putKV, fetchTraces } from '../lib/api'
import { useApp } from '../store'
import type { NodeId } from '../types'

export function KVConsole() {
  const nodes = useApp((s) => s.nodes)
  const toast = useApp((s) => s.toast)
  const setTraces = useApp((s) => s.setTraces)
  const setSelected = useApp((s) => s.setSelectedTrace)
  const [key, setKey] = useState('app/timeout')
  const [value, setValue] = useState('1500')
  const [out, setOut] = useState('等待操作')

  const leader = (Object.values(nodes).find((n) => n?.state === 'leader' && !n.dead)?.id || 'n1') as NodeId

  const write = async () => {
    if (!key.trim()) {
      toast('key 必填', true)
      return
    }
    try {
      const res = await putKV(leader, key, value)
      setOut(JSON.stringify(res, null, 2))
      if (res.trace_id) setSelected(res.trace_id)
      const traces = await fetchTraces(leader)
      setTraces(traces)
      toast(`写入成功 index=${res.index}`)
    } catch (e: any) {
      const addr = e.body?.leader
      toast(e.message || '写入失败', true)
      setOut(JSON.stringify(e.body || { error: e.message }, null, 2))
      if (addr) toast(`请改打 Leader ${addr}`)
    }
  }

  const read = async (stale: boolean) => {
    if (!key.trim()) {
      toast('key 必填', true)
      return
    }
    try {
      const res = await getKV(leader, key, stale)
      setOut(JSON.stringify(res, null, 2))
      toast(stale ? '脏读完成' : '线性一致读完成')
    } catch (e: any) {
      toast(e.message || '读取失败', true)
    }
  }

  return (
    <section className="rounded-xl border border-leader/40 bg-panel/80 p-3">
      <h2 className="mb-2 font-display text-sm text-leader">KV 控制台 · 线性一致</h2>
      <div className="mb-2 grid grid-cols-2 gap-2">
        <input aria-label="配置键" value={key} onChange={(e) => setKey(e.target.value)} className="rounded border border-grid bg-void px-2 py-1 text-sm" placeholder="key" />
        <input aria-label="配置值" value={value} onChange={(e) => setValue(e.target.value)} className="rounded border border-grid bg-void px-2 py-1 text-sm" placeholder="value" />
      </div>
      <div className="mb-2 flex flex-wrap gap-2">
        <button onClick={write} className="rounded bg-leader px-3 py-1 text-xs text-void">写入并追踪</button>
        <button onClick={() => read(false)} className="rounded border border-leader px-3 py-1 text-xs text-leader">线性读</button>
        <button onClick={() => read(true)} className="rounded border border-follower px-3 py-1 text-xs text-follower">脏读</button>
      </div>
      <pre className="max-h-28 overflow-auto font-mono text-[11px] text-mute">{out}</pre>
    </section>
  )
}
