import { useEffect } from 'react'
import { TopBar } from './components/TopBar'
import { Topology } from './components/Topology'
import { Waterfall } from './components/Waterfall'
import { Inspector } from './components/Inspector'
import { LogStream } from './components/LogStream'
import { ChaosPanel } from './components/ChaosPanel'
import { KVConsole } from './components/KVConsole'
import { ReplayBar } from './components/ReplayBar'
import { DialogHost, ToastHost } from './components/Dialog'
import { fetchAllStates, fetchTraces, openWS } from './lib/api'
import { useApp } from './store'
import { NODES, type NodeId } from './types'

export default function App() {
  const setNodes = useApp((s) => s.setNodes)
  const pushEvent = useApp((s) => s.pushEvent)
  const setTraces = useApp((s) => s.setTraces)

  useEffect(() => {
    let alive = true
    const tick = async () => {
      const st = await fetchAllStates()
      if (alive) setNodes(st)
      const leader = Object.values(st).find((n) => n?.state === 'leader' && !n.dead)
      if (leader) {
        const traces = await fetchTraces(leader.id as NodeId)
        if (alive) setTraces(traces)
      }
    }
    tick()
    const id = setInterval(tick, 800)
    const sockets = NODES.map((n) => {
      const connect = () => {
        const ws = openWS(n, pushEvent)
        ws.onclose = () => { if (alive) setTimeout(connect, 1500) }
        return ws
      }
      return connect()
    })
    return () => {
      alive = false
      clearInterval(id)
      sockets.forEach((s) => s.close())
    }
  }, [pushEvent, setNodes, setTraces])

  return (
    <div className="flex h-full w-full flex-col">
      <TopBar />
      <main className="grid min-h-0 flex-1 grid-cols-1 gap-3 p-3 lg:grid-cols-[56%_44%]">
        <Topology />
        <div className="flex min-h-0 flex-col gap-3">
          <Inspector />
          <div className="min-h-0 flex-1">
            <Waterfall />
          </div>
        </div>
      </main>
      <div className="grid grid-cols-1 gap-3 px-3 pb-3 lg:grid-cols-3">
        <KVConsole />
        <ChaosPanel />
        <div className="h-56">
          <LogStream />
        </div>
      </div>
      <ReplayBar />
      <DialogHost />
      <ToastHost />
    </div>
  )
}
