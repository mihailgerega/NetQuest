"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { StatusPill } from "@/components/ui/status-pill";
import type { Quest, QuestDifficulty } from "@/lib/api/types";
import { authedClient, getStoredUser } from "@/lib/auth";

const difficultyLabels: Record<QuestDifficulty, string> = {
  easy: "простое",
  medium: "среднее",
  hard: "сложное"
};

export default function QuestDetailPage() {
  const params = useParams<{ questId: string }>();
  const router = useRouter();
  const api = useMemo(() => authedClient(), []);
  const [quest, setQuest] = useState<Quest | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [revealedCount, setRevealedCount] = useState(0);

  useEffect(() => {
    if (!getStoredUser()) {
      router.push("/login");
      return;
    }
    api
      .getQuest(params.questId)
      .then((res) => setQuest(res.quest))
      .catch((error) => setMessage(error instanceof Error ? error.message : "Упражнение не найдено."));
  }, [api, params.questId, router]);

  async function startQuest() {
    if (!quest) return;
    setBusy(true);
    setMessage(null);
    try {
      const res = await api.startQuest(quest.id);
      let attempt = res.attempt;
      const openedHints = Math.min(revealedCount, res.quest.progressiveHints?.length ?? 0);
      if (openedHints > 0) {
        const hintRes = await api.revealQuestHint(res.attempt.id, { revealedHintsCount: openedHints });
        attempt = hintRes.attempt;
      }
      window.localStorage.setItem(`netquest.quest.${attempt.id}`, JSON.stringify({ quest: res.quest, attempt }));
      if (openedHints > 0) {
        window.localStorage.setItem(`netquest.quest.${attempt.id}.revealedHints`, String(openedHints));
      }
      router.push(`/simulator?questId=${encodeURIComponent(res.quest.id)}&attemptId=${encodeURIComponent(attempt.id)}`);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Не удалось начать упражнение.");
    } finally {
      setBusy(false);
    }
  }

  if (!quest) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-ink-950 px-6 text-white">
        <p className="rounded-md border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-slate-300">
          {message ?? "Загрузка упражнения..."}
        </p>
      </main>
    );
  }

  const progressiveHints = quest.progressiveHints ?? [];
  const visibleHints = progressiveHints.slice(0, revealedCount);
  const hasMoreHints = revealedCount < progressiveHints.length;

  return (
    <main className="min-h-screen bg-ink-950 px-6 py-8 text-white">
      <div className="mx-auto max-w-6xl space-y-6">
        <header className="border-b border-white/10 pb-6">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <p className="text-sm font-semibold text-signal-cyan">Quest Mode</p>
              <h1 className="mt-2 text-3xl font-bold">{quest.title}</h1>
              <p className="mt-2 text-sm text-slate-400">
                {quest.category} · {quest.estimatedMinutes} мин
              </p>
            </div>
            <StatusPill status={quest.difficulty === "easy" ? "healthy" : quest.difficulty === "medium" ? "degraded" : "down"} label={difficultyLabels[quest.difficulty]} />
          </div>
          <p className="mt-5 max-w-4xl leading-7 text-slate-300">{quest.description}</p>
        </header>

        {message && <p className="rounded-md border border-signal-amber/30 bg-signal-amber/10 px-4 py-3 text-sm text-signal-amber">{message}</p>}

        <section className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
          <article className="rounded-md border border-white/10 bg-white/[0.04] p-5">
            <h2 className="font-bold">Цель</h2>
            <p className="mt-3 leading-7 text-slate-300">{quest.goal}</p>

            <h3 className="mt-6 font-bold">Что вы поймёте</h3>
            <div className="mt-3 grid gap-2 sm:grid-cols-2">
              {quest.learningObjectives.map((objective) => (
                <div className="rounded-md border border-white/10 bg-ink-950 px-3 py-2 text-sm text-slate-300" key={objective}>
                  {objective}
                </div>
              ))}
            </div>

            <h3 className="mt-6 font-bold">Как сервер проверит решение</h3>
            <p className="mt-2 text-sm leading-6 text-slate-400">
              После запуска проверки backend получит текущую топологию, выполнит симуляцию и сравнит фактическое поведение сети с критериями ниже.
            </p>
            <div className="mt-3 space-y-2">
              {quest.expectedChecks.map((check, index) => (
                <div className="rounded-md border border-white/10 bg-ink-950 px-3 py-2 text-sm text-slate-300" key={check.id}>
                  <span className="text-slate-500">{index + 1}.</span> {check.title}
                </div>
              ))}
            </div>

            {quest.realWorldImportance && (
              <div className="mt-6 rounded-md border border-signal-cyan/25 bg-signal-cyan/10 p-4">
                <h3 className="font-bold text-signal-cyan">Почему это важно</h3>
                <p className="mt-2 text-sm leading-6 text-slate-200">{quest.realWorldImportance}</p>
              </div>
            )}
          </article>

          <aside className="space-y-4">
            <div className="rounded-md border border-white/10 bg-white/[0.04] p-5">
              <h2 className="font-bold">Постепенные подсказки</h2>
              <p className="mt-2 text-sm leading-6 text-slate-400">
                Подсказки открываются по одной. Сначала они помогают понять слой проблемы, затем подводят к конкретному действию.
              </p>

              <div className="mt-4 space-y-3">
                {visibleHints.map((hint, index) => (
                  <div className="rounded-md border border-white/10 bg-ink-950 p-3 text-sm" key={`${hint.title}-${index}`}>
                    <p className="font-semibold text-slate-100">{hint.title}</p>
                    <p className="mt-2 leading-6 text-slate-300">{hint.body}</p>
                    {(hint.actions ?? []).length > 0 && (
                      <div className="mt-3 space-y-1 text-xs text-slate-400">
                        {hint.actions?.map((action) => <p key={action}>- {action}</p>)}
                      </div>
                    )}
                  </div>
                ))}
                {visibleHints.length === 0 && (
                  <p className="rounded-md border border-white/10 bg-ink-950 p-3 text-sm text-slate-400">
                    Попробуйте сначала запустить симуляцию и найти первое событие с ошибкой в Timeline.
                  </p>
                )}
              </div>

              <Button className="mt-4 w-full" variant="secondary" onClick={() => setRevealedCount((count) => Math.min(count + 1, progressiveHints.length))} disabled={!hasMoreHints}>
                {hasMoreHints ? `Показать подсказку ${revealedCount + 1}` : "Все подсказки открыты"}
              </Button>
            </div>

            {(quest.glossaryTerms ?? []).length > 0 && (
              <div className="rounded-md border border-white/10 bg-white/[0.04] p-5">
                <h2 className="font-bold">Мини-глоссарий</h2>
                <div className="mt-3 space-y-3 text-sm">
                  {quest.glossaryTerms.map((item) => (
                    <div key={item.term}>
                      <p className="font-semibold text-slate-100">{item.term}</p>
                      <p className="mt-1 leading-6 text-slate-400">{item.definition}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="rounded-md border border-white/10 bg-white/[0.04] p-5">
              <Button className="w-full" onClick={startQuest} disabled={busy}>
                {busy ? "Запускаю..." : "Начать в симуляторе"}
              </Button>
              <Button className="mt-3 w-full" href="/quests" variant="secondary">
                Все упражнения
              </Button>
            </div>
          </aside>
        </section>
      </div>
    </main>
  );
}
