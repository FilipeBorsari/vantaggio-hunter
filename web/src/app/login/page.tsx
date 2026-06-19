"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export default function LoginPage() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError("");
    setLoading(true);

    const fd = new FormData(e.currentTarget);
    const email = fd.get("email") as string;
    const password = fd.get("password") as string;

    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        setError(body?.error ?? "Credenciais inválidas");
        return;
      }
      const data = await res.json().catch(() => ({}));
      router.push(data.redirect ?? "/companies");
      router.refresh();
    } catch {
      setError("Erro ao conectar com o servidor");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-v-bg">
      <div className="w-full max-w-md">
        <div className="bg-v-card rounded-2xl border border-v-border p-8">
          <div className="mb-8 text-center">
            <h1 className="text-2xl font-bold text-v-accent">Vantaggio Hunter</h1>
            <p className="mt-1 text-sm text-v-muted">Faça login para continuar</p>
          </div>

          <form onSubmit={handleSubmit} method="POST" className="space-y-5">
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-v-text/80 mb-1">
                E-mail
              </label>
              <input
                id="email"
                name="email"
                type="email"
                required
                autoComplete="email"
                className="w-full px-3 py-2 border border-v-border rounded-lg text-sm text-v-text bg-v-bg placeholder:text-v-muted focus:outline-none focus:ring-2 focus:ring-v-accent focus:border-transparent"
                placeholder="voce@empresa.com"
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-v-text/80 mb-1">
                Senha
              </label>
              <input
                id="password"
                name="password"
                type="password"
                required
                autoComplete="current-password"
                className="w-full px-3 py-2 border border-v-border rounded-lg text-sm text-v-text bg-v-bg placeholder:text-v-muted focus:outline-none focus:ring-2 focus:ring-v-accent focus:border-transparent"
                placeholder="••••••••"
              />
            </div>

            {error && (
              <p className="text-sm text-red-400 bg-red-900/30 border border-red-900/50 rounded-lg px-3 py-2">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-v-accent hover:bg-v-glow disabled:opacity-50 text-white font-medium py-2 px-4 rounded-lg text-sm transition-colors"
            >
              {loading ? "Entrando..." : "Entrar"}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
