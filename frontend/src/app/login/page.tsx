"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { createApiClient } from "@/lib/api/client";
import { storeAuth } from "@/lib/auth";

export default function LoginPage() {
  const router = useRouter();
  const api = createApiClient();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("demo@netquest.local");
  const [password, setPassword] = useState("very-secure-password");
  const [displayName, setDisplayName] = useState("Пользователь NetQuest");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setMessage(null);
    try {
      const response =
        mode === "login"
          ? await api.login({ email, password })
          : await api.register({ email, password, displayName });
      storeAuth(response);
      router.push("/dashboard");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Не удалось войти. Проверьте email и пароль.");
    } finally {
      setLoading(false);
    }
  }

  async function demoLogin() {
    setLoading(true);
    setMessage(null);
    try {
      const response = await api.demoLogin();
      storeAuth(response);
      router.push("/dashboard");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Не удалось открыть демо-вход.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-ink-950 px-6 text-white">
      <section className="w-full max-w-md rounded-md border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-black/30">
        <p className="text-sm font-semibold text-signal-cyan">NetQuest</p>
        <h1 className="mt-3 text-3xl font-bold">{mode === "login" ? "Вход" : "Создать аккаунт"}</h1>
        <p className="mt-3 text-sm leading-6 text-slate-400">
          Аккаунт нужен, чтобы сохранять проекты, запускать Quest Mode и возвращаться к своим топологиям. Для быстрого просмотра можно открыть демо.
        </p>
        <form className="mt-8 space-y-4" onSubmit={submit}>
          {mode === "register" && (
            <label className="block text-sm">
              <span className="text-slate-300">Имя</span>
              <input className="mt-2 h-11 w-full rounded-md border border-white/10 bg-ink-900 px-3 text-white outline-none focus:border-signal-cyan" value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
            </label>
          )}
          <label className="block text-sm">
            <span className="text-slate-300">Email</span>
            <input className="mt-2 h-11 w-full rounded-md border border-white/10 bg-ink-900 px-3 text-white outline-none focus:border-signal-cyan" type="email" value={email} onChange={(event) => setEmail(event.target.value)} />
          </label>
          <label className="block text-sm">
            <span className="text-slate-300">Пароль</span>
            <input className="mt-2 h-11 w-full rounded-md border border-white/10 bg-ink-900 px-3 text-white outline-none focus:border-signal-cyan" type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
          </label>
          {message && <p className="rounded-md border border-signal-red/30 bg-signal-red/10 px-3 py-2 text-sm text-signal-red">{message}</p>}
          <Button className="w-full" disabled={loading} type="submit">
            {loading ? "Работаю..." : mode === "login" ? "Продолжить" : "Зарегистрироваться"}
          </Button>
          <Button className="w-full" disabled={loading} onClick={demoLogin} variant="secondary">
            Открыть демо
          </Button>
        </form>
        <button className="mt-5 text-sm text-slate-300 hover:text-white" onClick={() => setMode(mode === "login" ? "register" : "login")}>
          {mode === "login" ? "Создать локальный аккаунт" : "Использовать существующий аккаунт"}
        </button>
      </section>
    </main>
  );
}
