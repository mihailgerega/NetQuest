const events = [
  "[0ms] симуляция запущена",
  "[2ms] топология проверена",
  "[3ms] пакет создан",
  "[76ms] симуляция завершена"
];

export function Timeline() {
  return (
    <footer className="h-44 border-t border-white/10 bg-ink-900/95">
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-between border-b border-white/10 px-4 py-3 text-sm font-semibold">
          <span>Timeline событий</span>
          <span className="text-xs text-slate-400">x1</span>
        </div>
        <div className="grid gap-2 overflow-auto p-3 font-mono text-xs text-slate-300 md:grid-cols-2">
          {events.map((event) => (
            <div className="rounded-md border border-white/10 bg-white/[0.04] px-3 py-2" key={event}>
              {event}
            </div>
          ))}
        </div>
      </div>
    </footer>
  );
}
