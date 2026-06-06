"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { StatusPill } from "@/components/ui/status-pill";
import type { Quest, QuestDifficulty } from "@/lib/api/types";
import { authedClient, getStoredUser } from "@/lib/auth";

const difficultyLabels: Record<QuestDifficulty, string> = {
  easy: "простое",
  medium: "среднее",
  hard: "сложное"
};

const difficultyPluralLabels: Record<QuestDifficulty, string> = {
  easy: "Простые",
  medium: "Средние",
  hard: "Сложные"
};

const difficultyStatus: Record<QuestDifficulty, "healthy" | "degraded" | "down"> = {
  easy: "healthy",
  medium: "degraded",
  hard: "down"
};

const difficultyOrder: Record<QuestDifficulty, number> = {
  easy: 0,
  medium: 1,
  hard: 2
};

const preferredCategoryOrder = ["DNS", "Routing", "Firewall", "Load Balancer", "Latency", "Failover", "Security", "TLS"];

type DifficultyFilter = "all" | QuestDifficulty;
type TopicFilter = "all" | string;

const difficultyFilters: Array<{ value: DifficultyFilter; label: string }> = [
  { value: "all", label: "Все сложности" },
  { value: "easy", label: "Простые" },
  { value: "medium", label: "Средние" },
  { value: "hard", label: "Сложные" }
];

export default function QuestsPage() {
  const router = useRouter();
  const api = useMemo(() => authedClient(), []);
  const [quests, setQuests] = useState<Quest[]>([]);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState<string | null>(null);
  const [difficultyFilter, setDifficultyFilter] = useState<DifficultyFilter>("all");
  const [topicFilter, setTopicFilter] = useState<TopicFilter>("all");

  useEffect(() => {
    if (!getStoredUser()) {
      router.push("/login");
      return;
    }
    api
      .listQuests()
      .then((res) => setQuests(res.quests))
      .catch((error) => setMessage(error instanceof Error ? error.message : "Не удалось загрузить упражнения. Проверьте соединение с API."))
      .finally(() => setLoading(false));
  }, [api, router]);

  const completed = quests.filter((quest) => quest.attemptStatus === "completed").length;
  const categories = useMemo(() => {
    const unique = Array.from(new Set(quests.map((quest) => quest.category).filter(Boolean)));
    return unique.sort(compareCategories);
  }, [quests]);

  const progressByDifficulty = useMemo(() => {
    return (["easy", "medium", "hard"] as QuestDifficulty[]).map((difficulty) => {
      const total = quests.filter((quest) => quest.difficulty === difficulty).length;
      const done = quests.filter((quest) => quest.difficulty === difficulty && quest.attemptStatus === "completed").length;
      return { difficulty, total, done };
    });
  }, [quests]);

  const visibleQuests = useMemo(
    () =>
      quests
        .filter((quest) => difficultyFilter === "all" || quest.difficulty === difficultyFilter)
        .filter((quest) => topicFilter === "all" || quest.category === topicFilter)
        .sort(compareQuests),
    [difficultyFilter, quests, topicFilter]
  );

  return (
    <main className="min-h-screen bg-ink-950 px-6 py-8 text-white">
      <div className="mx-auto max-w-7xl space-y-6">
        <header className="flex flex-wrap items-center justify-between gap-4 border-b border-white/10 pb-6">
          <div>
            <p className="text-sm font-semibold text-signal-cyan">Quest Mode</p>
            <h1 className="mt-2 text-3xl font-bold">Упражнения NetQuest</h1>
            <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-400">
              Исправляйте сломанные топологии, запускайте симуляцию и проверяйте решение на сервере. Упражнения идут от простых сетевых ошибок к сложным multi-layer incidents.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <StatusPill status="running" label={`${completed}/${quests.length || 0} выполнено`} />
            <Button href="/dashboard" variant="secondary">
              Проекты
            </Button>
          </div>
        </header>

        <section className="grid gap-3 md:grid-cols-3">
          {progressByDifficulty.map((item) => (
            <div className="rounded-md border border-white/10 bg-white/[0.04] px-4 py-3" key={item.difficulty}>
              <div className="flex items-center justify-between gap-3">
                <span className="text-sm font-semibold">{difficultyPluralLabels[item.difficulty]}</span>
                <StatusPill status={difficultyStatus[item.difficulty]} label={`${item.done}/${item.total}`} />
              </div>
              <div className="mt-3 h-2 overflow-hidden rounded-full bg-white/10">
                <div className="h-full rounded-full bg-signal-cyan" style={{ width: `${item.total ? Math.round((item.done / item.total) * 100) : 0}%` }} />
              </div>
            </div>
          ))}
        </section>

        <section className="space-y-3 rounded-md border border-white/10 bg-white/[0.03] p-4">
          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Сложность</p>
            <div className="flex flex-wrap items-center gap-2">
              {difficultyFilters.map((item) => (
                <button
                  className={`min-h-9 rounded-md border px-3 text-sm font-semibold transition ${
                    difficultyFilter === item.value ? "border-signal-cyan bg-signal-cyan/15 text-signal-cyan" : "border-white/10 bg-white/[0.04] text-slate-300 hover:bg-white/[0.08]"
                  }`}
                  key={item.value}
                  onClick={() => setDifficultyFilter(item.value)}
                >
                  {item.label}
                </button>
              ))}
            </div>
          </div>

          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Тема</p>
            <div className="flex flex-wrap items-center gap-2">
              <button
                className={`min-h-9 rounded-md border px-3 text-sm font-semibold transition ${
                  topicFilter === "all" ? "border-signal-cyan bg-signal-cyan/15 text-signal-cyan" : "border-white/10 bg-white/[0.04] text-slate-300 hover:bg-white/[0.08]"
                }`}
                onClick={() => setTopicFilter("all")}
              >
                Все темы
              </button>
              {categories.map((category) => (
                <button
                  className={`min-h-9 rounded-md border px-3 text-sm font-semibold transition ${
                    topicFilter === category ? "border-signal-cyan bg-signal-cyan/15 text-signal-cyan" : "border-white/10 bg-white/[0.04] text-slate-300 hover:bg-white/[0.08]"
                  }`}
                  key={category}
                  onClick={() => setTopicFilter(category)}
                >
                  {category}
                </button>
              ))}
            </div>
          </div>
        </section>

        {message && <p className="rounded-md border border-signal-amber/30 bg-signal-amber/10 px-4 py-3 text-sm text-signal-amber">{message}</p>}

        <section className="grid gap-4 md:grid-cols-3">
          {loading && [1, 2, 3, 4, 5, 6].map((item) => <div className="h-56 animate-pulse rounded-md border border-white/10 bg-white/[0.04]" key={item} />)}
          {!loading &&
            visibleQuests.map((quest) => (
              <article className="flex min-h-56 flex-col rounded-md border border-white/10 bg-white/[0.04] p-5" key={quest.id}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h2 className="font-bold">{quest.title}</h2>
                    <p className="mt-1 text-xs text-slate-400">
                      {quest.category} · {quest.estimatedMinutes} мин
                    </p>
                  </div>
                  <StatusPill status={difficultyStatus[quest.difficulty]} label={difficultyLabels[quest.difficulty]} />
                </div>
                <p className="mt-4 flex-1 text-sm leading-6 text-slate-300">{quest.description}</p>
                <div className="mt-4 flex items-center justify-between gap-3">
                  <StatusPill status={quest.attemptStatus === "completed" ? "healthy" : quest.attemptStatus === "failed" ? "down" : quest.attemptStatus === "in_progress" ? "running" : "idle"} label={attemptLabel(quest.attemptStatus)} />
                  <Button href={`/quests/${quest.slug}`} className="min-h-9 px-3">
                    Открыть
                  </Button>
                </div>
              </article>
            ))}
          {!loading && visibleQuests.length === 0 && (
            <div className="rounded-md border border-white/10 bg-white/[0.04] p-5 text-sm text-slate-300 md:col-span-3">
              Для выбранных фильтров упражнений пока нет.
            </div>
          )}
        </section>
      </div>
    </main>
  );
}

function compareQuests(left: Quest, right: Quest) {
  return (
    difficultyOrder[left.difficulty] - difficultyOrder[right.difficulty] ||
    compareCategories(left.category, right.category) ||
    left.estimatedMinutes - right.estimatedMinutes ||
    left.title.localeCompare(right.title, "ru")
  );
}

function compareCategories(left: string, right: string) {
  const leftIndex = preferredCategoryOrder.indexOf(left);
  const rightIndex = preferredCategoryOrder.indexOf(right);
  const normalizedLeft = leftIndex === -1 ? Number.MAX_SAFE_INTEGER : leftIndex;
  const normalizedRight = rightIndex === -1 ? Number.MAX_SAFE_INTEGER : rightIndex;
  return normalizedLeft - normalizedRight || left.localeCompare(right, "ru");
}

function attemptLabel(status?: string) {
  if (status === "completed") return "выполнено";
  if (status === "failed") return "нужна правка";
  if (status === "in_progress") return "в процессе";
  return "не начато";
}
