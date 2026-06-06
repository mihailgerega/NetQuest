"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Panel } from "@/components/ui/panel";
import { StatusPill } from "@/components/ui/status-pill";
import type { Project, User } from "@/lib/api/types";
import { authedClient, getStoredUser } from "@/lib/auth";

export default function DashboardPage() {
  const router = useRouter();
  const api = useMemo(() => authedClient(), []);
  const [projects, setProjects] = useState<Project[]>([]);
  const [name, setName] = useState("NetQuest Demo Lab");
  const [message, setMessage] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    const stored = getStoredUser();
    if (!stored) {
      router.push("/login");
      return;
    }
    setUser(stored);
    api
      .listProjects()
      .then((res) => setProjects(res.projects))
      .catch((error) => setMessage(error instanceof Error ? error.message : "Не удалось загрузить проекты. Проверьте, что backend запущен."))
      .finally(() => setLoading(false));
  }, [api, router]);

  async function createProject(event?: FormEvent) {
    event?.preventDefault();
    setMessage(null);
    try {
      const res = await api.createProject({ name, visibility: "private" });
      setProjects((items) => [res.project, ...items]);
      router.push(`/simulator?projectId=${res.project.id}`);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Не удалось создать проект.")
    }
  }

  return (
    <main className="min-h-screen bg-ink-950 px-6 py-8 text-white">
      <div className="mx-auto flex max-w-7xl flex-col gap-6">
        <header className="flex flex-wrap items-center justify-between gap-4 border-b border-white/10 pb-6">
          <div>
            <p className="text-sm font-semibold text-signal-cyan">NetQuest</p>
            <h1 className="mt-2 text-3xl font-bold">Проекты</h1>
            <p className="mt-2 text-sm text-slate-400">{user ? user.email : "Проверяю сессию..."}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button href="/quests" variant="secondary">
              Quest Mode
            </Button>
            <Button onClick={() => createProject()} disabled={loading}>
              Новая сеть
            </Button>
          </div>
        </header>

        {message && <p className="rounded-md border border-signal-amber/30 bg-signal-amber/10 px-4 py-3 text-sm text-signal-amber">{message}</p>}

        <Panel className="rounded-md p-5">
          <form className="flex flex-col gap-3 md:flex-row" onSubmit={createProject}>
            <input className="h-11 flex-1 rounded-md border border-white/10 bg-ink-900 px-3 text-sm outline-none focus:border-signal-cyan" value={name} onChange={(event) => setName(event.target.value)} />
            <Button type="submit" disabled={loading}>
              Создать проект
            </Button>
          </form>
        </Panel>

        <section className="grid gap-4 md:grid-cols-3">
          {loading &&
            [1, 2, 3].map((item) => <div className="h-32 animate-pulse rounded-md border border-white/10 bg-white/[0.04]" key={item} />)}
          {!loading && projects.length === 0 && (
            <Panel className="rounded-md p-6 md:col-span-3">
              <h2 className="text-lg font-semibold">Проектов пока нет</h2>
              <p className="mt-2 text-sm text-slate-400">Создайте проект, загрузите демо-топологию и запустите первую симуляцию.</p>
            </Panel>
          )}
          {projects.map((project) => (
            <button className="text-left" key={project.id} onClick={() => router.push(`/simulator?projectId=${project.id}`)}>
              <Panel className="h-full rounded-md p-5 transition hover:border-signal-cyan/50 hover:bg-signal-cyan/5">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2 className="font-semibold">{project.name}</h2>
                    <p className="mt-2 text-sm text-slate-400">{project.visibility}</p>
                  </div>
                  <StatusPill status="healthy" />
                </div>
              </Panel>
            </button>
          ))}
        </section>
      </div>
    </main>
  );
}
