import { StatusPill } from "@/components/ui/status-pill";

export function Inspector() {
  return (
    <aside className="flex h-full min-h-0 flex-col border-l border-white/10 bg-ink-900/95">
      <div className="border-b border-white/10 px-4 py-3 text-sm font-semibold">Инспектор</div>
      <div className="space-y-5 overflow-auto p-4 text-sm">
        <div>
          <p className="text-xs uppercase text-slate-500">Выбрано</p>
          <h2 className="mt-2 text-lg font-semibold">Firewall-1</h2>
          <div className="mt-3">
            <StatusPill status="degraded" />
          </div>
        </div>
        <div className="rounded-md border border-white/10 bg-white/[0.04] p-3">
          <p className="text-xs text-slate-400">Rule #100</p>
          <p className="mt-2 text-slate-200">allow tcp 10.0.1.0/24 to 10.0.2.10:443</p>
        </div>
        <div className="rounded-md border border-white/10 bg-white/[0.04] p-3">
          <p className="text-xs text-slate-400">Метрики</p>
          <p className="mt-2 text-slate-200">latency p95 42ms</p>
        </div>
      </div>
    </aside>
  );
}
