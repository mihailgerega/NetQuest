const nodes = ["Client", "Server", "Router", "Switch", "DNS", "Firewall", "Load Balancer", "Proxy", "NAT", "VPN", "Database", "Internet"];

export function NodePalette() {
  return (
    <aside className="flex min-h-0 flex-col border-r border-white/10 bg-ink-900/95">
      <div className="border-b border-white/10 px-4 py-3 text-sm font-semibold">Узлы</div>
      <div className="grid gap-2 overflow-auto p-3">
        {nodes.map((node) => (
          <button
            className="flex h-11 items-center justify-between rounded-md border border-white/10 bg-white/[0.04] px-3 text-left text-sm text-slate-200 transition hover:border-signal-cyan/50 hover:bg-signal-cyan/10"
            key={node}
          >
            <span>{node}</span>
            <span className="h-2 w-2 rounded-full bg-signal-cyan" />
          </button>
        ))}
      </div>
    </aside>
  );
}
