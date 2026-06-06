const nodes = [
  { id: "Client", className: "left-[8%] top-[54%] border-signal-cyan text-signal-cyan" },
  { id: "DNS", className: "left-[25%] top-[24%] border-signal-blue text-signal-blue" },
  { id: "Firewall", className: "left-[42%] top-[50%] border-signal-amber text-signal-amber" },
  { id: "LB", className: "left-[60%] top-[32%] border-signal-green text-signal-green" },
  { id: "API-1", className: "right-[11%] top-[18%] border-white/35 text-white" },
  { id: "API-2", className: "right-[7%] top-[52%] border-white/35 text-white" }
];

export function NetworkPreview() {
  return (
    <div className="pointer-events-none absolute inset-0 overflow-hidden opacity-90">
      <div className="network-grid absolute inset-0" />
      <div className="absolute inset-x-0 top-20 mx-auto h-[360px] max-w-5xl">
        <div className="absolute left-[12%] top-[59%] h-px w-[19%] rotate-[-28deg] bg-signal-cyan/40" />
        <div className="absolute left-[29%] top-[31%] h-px w-[19%] rotate-[23deg] bg-signal-blue/40" />
        <div className="absolute left-[45%] top-[56%] h-px w-[18%] rotate-[-22deg] bg-signal-amber/40" />
        <div className="absolute right-[18%] top-[37%] h-px w-[18%] rotate-[-18deg] bg-signal-green/40" />
        <div className="absolute right-[15%] top-[48%] h-px w-[15%] rotate-[28deg] bg-white/20" />
        <span className="packet-flight absolute left-[12%] top-[60%] h-3 w-3 rounded-full bg-signal-cyan shadow-glow" />
        {nodes.map((node) => (
          <div
            className={`node-pulse absolute flex h-16 w-24 items-center justify-center rounded-md border bg-ink-900/90 text-xs font-bold tracking-wide ${node.className}`}
            key={node.id}
          >
            {node.id}
          </div>
        ))}
      </div>
    </div>
  );
}
