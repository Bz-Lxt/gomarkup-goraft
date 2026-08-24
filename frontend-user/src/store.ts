import { create } from 'zustand'
import type { NodeStatus, RaftEvent, Trace } from './types'
import { NODES } from './types'

interface Toast {
  id: number
  text: string
  danger?: boolean
}

interface AppState {
  nodes: Record<string, NodeStatus | null>
  events: RaftEvent[]
  traces: Trace[]
  paused: boolean
  speed: number
  cursorUs: number
  selectedTrace: string | null
  toasts: Toast[]
  confirm: { title: string; body: string; action: () => Promise<void> } | null
  setNodes: (n: Record<string, NodeStatus | null>) => void
  pushEvent: (ev: RaftEvent) => void
  setTraces: (t: Trace[]) => void
  setPaused: (v: boolean) => void
  setSpeed: (v: number) => void
  setCursor: (v: number) => void
  setSelectedTrace: (id: string | null) => void
  toast: (text: string, danger?: boolean) => void
  dismiss: (id: number) => void
  ask: (title: string, body: string, action: () => Promise<void>) => void
  closeAsk: () => void
}

let toastSeq = 1

export const useApp = create<AppState>((set) => ({
  nodes: Object.fromEntries(NODES.map((id) => [id, null])),
  events: [],
  traces: [],
  paused: false,
  speed: 1,
  cursorUs: 0,
  selectedTrace: null,
  toasts: [],
  confirm: null,
  setNodes: (nodes) => set({ nodes }),
  pushEvent: (ev) => set((s) => {
    if (s.paused) return s
    const events = [...s.events, ev]
    return { events: events.slice(-8000), cursorUs: ev.ts_unix_us }
  }),
  setTraces: (traces) => set({ traces }),
  setPaused: (paused) => set({ paused }),
  setSpeed: (speed) => set({ speed }),
  setCursor: (cursorUs) => set({ cursorUs }),
  setSelectedTrace: (selectedTrace) => set({ selectedTrace }),
  toast: (text, danger) => {
    const id = toastSeq++
    set((s) => ({ toasts: [...s.toasts, { id, text, danger }] }))
    setTimeout(() => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })), 5000)
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
  ask: (title, body, action) => set({ confirm: { title, body, action } }),
  closeAsk: () => set({ confirm: null }),
}))
