import { useApp } from '../store'

export function DialogHost() {
  const confirm = useApp((s) => s.confirm)
  const close = useApp((s) => s.closeAsk)
  const toast = useApp((s) => s.toast)
  if (!confirm) return null
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div className="w-full max-w-md rounded-xl border border-danger/50 bg-panel p-5">
        <h3 className="font-display text-lg text-danger">{confirm.title}</h3>
        <p className="mt-2 text-sm text-mute">{confirm.body}</p>
        <div className="mt-4 flex justify-end gap-2">
          <button onClick={close} className="rounded border border-grid px-3 py-1 text-sm">取消</button>
          <button
            onClick={async () => {
              try { await confirm.action() } catch (e: any) { toast(e.message || '操作失败', true) }
              close()
            }}
            className="rounded bg-danger px-3 py-1 text-sm text-void"
          >
            确认
          </button>
        </div>
      </div>
    </div>
  )
}

export function ToastHost() {
  const toasts = useApp((s) => s.toasts)
  const dismiss = useApp((s) => s.dismiss)
  return (
    <div className="fixed right-4 top-4 z-40 flex flex-col gap-2">
      {toasts.map((t) => (
        <div key={t.id} className={`toast-enter flex items-center gap-3 rounded border px-3 py-2 text-sm ${t.danger ? 'border-danger text-danger' : 'border-follower text-follower'} bg-panel`}>
          <span>{t.text}</span>
          <button aria-label="关闭提示" onClick={() => dismiss(t.id)}>×</button>
        </div>
      ))}
    </div>
  )
}
