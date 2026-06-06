import { cn } from "@/lib/utils";

type Status = "healthy" | "degraded" | "down" | "running" | "idle";

const styles: Record<Status, string> = {
  healthy: "border-signal-green/40 bg-signal-green/10 text-signal-green",
  degraded: "border-signal-amber/40 bg-signal-amber/10 text-signal-amber",
  down: "border-signal-red/40 bg-signal-red/10 text-signal-red",
  running: "border-signal-cyan/40 bg-signal-cyan/10 text-signal-cyan",
  idle: "border-slate-500/40 bg-slate-500/10 text-slate-300"
};

export function StatusPill({ status, label }: { status: Status; label?: string }) {
  return (
    <span className={cn("inline-flex h-7 items-center rounded-md border px-2.5 text-xs font-semibold", styles[status])}>
      {label ?? status}
    </span>
  );
}
