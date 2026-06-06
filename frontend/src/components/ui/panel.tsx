import { cn } from "@/lib/utils";

type PanelProps = {
  children: React.ReactNode;
  className?: string;
};

export function Panel({ children, className }: PanelProps) {
  return (
    <section className={cn("border border-white/10 bg-white/[0.04] shadow-2xl shadow-black/20", className)}>
      {children}
    </section>
  );
}
