export function CanvasPlaceholder() {
  return (
    <section className="network-grid relative min-h-0 overflow-hidden bg-ink-950">
      <div className="absolute left-[16%] top-[38%] h-14 w-24 rounded-md border border-signal-cyan bg-ink-900 text-center text-xs font-bold leading-[3.5rem] text-signal-cyan shadow-glow">
        Client
      </div>
      <div className="absolute left-[42%] top-[28%] h-14 w-24 rounded-md border border-signal-amber bg-ink-900 text-center text-xs font-bold leading-[3.5rem] text-signal-amber">
        Firewall
      </div>
      <div className="absolute right-[18%] top-[42%] h-14 w-24 rounded-md border border-signal-green bg-ink-900 text-center text-xs font-bold leading-[3.5rem] text-signal-green">
        Server
      </div>
      <div className="absolute left-[25%] top-[45%] h-px w-[22%] rotate-[-15deg] bg-signal-cyan/50" />
      <div className="absolute right-[28%] top-[42%] h-px w-[23%] rotate-[12deg] bg-signal-green/45" />
      <span className="absolute left-[38%] top-[38%] h-3 w-3 rounded-full bg-signal-cyan shadow-glow" />
      <div className="absolute bottom-5 left-5 rounded-md border border-white/10 bg-black/35 px-3 py-2 text-xs text-slate-300">
        Черновик сетевой топологии
      </div>
    </section>
  );
}
