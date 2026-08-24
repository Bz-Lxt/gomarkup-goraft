import { useEffect, useRef } from 'react'
import * as d3 from 'd3'
import { useApp } from '../store'
import { NODES } from '../types'

const COLOR: Record<string, string> = {
  leader: '#e8b84a',
  follower: '#3ee0c6',
  candidate: '#ff8a3d',
  learner: '#7dd3fc',
  unknown: '#6b7280',
}

export function Topology() {
  const ref = useRef<SVGSVGElement | null>(null)
  const nodes = useApp((s) => s.nodes)
  const events = useApp((s) => s.events)

  useEffect(() => {
    const svg = d3.select(ref.current)
    const wrap = ref.current?.parentElement
    if (!wrap || !ref.current) return
    const width = wrap.clientWidth
    const height = wrap.clientHeight
    svg.attr('viewBox', `0 0 ${width} ${height}`)

    const items = NODES.map((id) => {
      const st = nodes[id]
      return {
        id,
        state: st?.dead ? 'unknown' : (st?.state ?? 'unknown'),
        term: st?.term ?? 0,
        dead: !!st?.dead,
      }
    })

    const sim = d3.forceSimulation(items as any)
      .force('charge', d3.forceManyBody().strength(-280))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collide', d3.forceCollide(56))
      .stop()
    for (let i = 0; i < 80; i++) sim.tick()

    svg.selectAll('*').remove()
    const g = svg.append('g')

    const recentVotes = events.slice(-40).filter((e) => e.type === 'vote_request' || e.type === 'vote_granted' || e.type === 'heartbeat' || e.type === 'append_send')
    recentVotes.forEach((ev) => {
      const from = items.find((n) => n.id === ev.node_id)
      const toId = String(ev.payload?.to || ev.payload?.from || '')
      const to = items.find((n) => n.id === toId)
      if (!from || !to) return
      g.append('line')
        .attr('x1', (from as any).x).attr('y1', (from as any).y)
        .attr('x2', (to as any).x).attr('y2', (to as any).y)
        .attr('stroke', ev.type.includes('vote') ? '#ff8a3d' : '#3ee0c6')
        .attr('stroke-width', 1.2)
        .attr('opacity', 0.55)
    })

    const node = g.selectAll('g.node').data(items).enter().append('g')
      .attr('class', 'node')
      .attr('transform', (d: any) => `translate(${d.x},${d.y})`)

    node.append('circle')
      .attr('r', (d) => d.state === 'leader' ? 28 : 22)
      .attr('fill', '#10151d')
      .attr('stroke', (d) => d.dead ? '#6b7280' : COLOR[d.state])
      .attr('stroke-width', 3)

    node.filter((d) => d.state === 'leader' && !d.dead).append('circle')
      .attr('r', 36)
      .attr('fill', 'none')
      .attr('stroke', COLOR.leader)
      .attr('stroke-opacity', 0.35)
      .attr('class', 'pulse')

    node.append('text')
      .attr('text-anchor', 'middle')
      .attr('dy', 4)
      .attr('fill', '#d7e0ea')
      .attr('font-family', 'Chakra Petch')
      .attr('font-size', 13)
      .text((d) => d.id.toUpperCase())

    node.append('text')
      .attr('text-anchor', 'middle')
      .attr('dy', 42)
      .attr('fill', '#7f8b99')
      .attr('font-family', 'IBM Plex Mono')
      .attr('font-size', 10)
      .text((d) => d.dead ? 'DOWN' : d.state.toUpperCase())
  }, [nodes, events])

  return (
    <div className="scan-grid relative h-full w-full overflow-hidden rounded-xl border border-grid bg-panel/80">
      <svg ref={ref} className="h-full w-full" role="img" aria-label="集群拓扑" />
    </div>
  )
}
