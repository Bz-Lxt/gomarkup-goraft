import { useApp } from '../store'

export function ReplayBar() {
  const events = useApp((s) => s.events)
  const paused = useApp((s) => s.paused)
  const setPaused = useApp((s) => s.setPaused)
  const speed = useApp((s) => s.speed)
  const setSpeed = useApp((s) => s.setSpeed)
  const cursor = useApp((s) => s.cursorUs)
  const setCursor = useApp((s) => s.setCursor)
  const min = events[0]?.ts_unix_us ?? 0
  const max = events[events.length - 1]?.ts_unix_us ?? 1

  return (
    <div className="flex w-full items-center gap-3 border-t border-grid px-4 py-2">
      <button onClick={() => setPaused(!paused)} className="rounded border border-grid px-2 py-1 text-xs">{paused ? '播放' : '暂停回放'}</button>
      <select aria-label="回放倍速" value={speed} onChange={(e) => setSpeed(Number(e.target.value))} className="rounded border border-grid bg-void px-2 py-1 text-xs">
        <option value={0.25}>0.25x</option>
        <option value={0.5}>0.5x</option>
        <option value={1}>1x</option>
      </select>
      <input
        aria-label="时间轴"
        type="range"
        min={min}
        max={max}
        value={cursor || max}
        onChange={(e) => { setPaused(true); setCursor(Number(e.target.value)) }}
        className="w-full"
      />
    </div>
  )
}
