"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";

interface InvitationInfo {
  email: string;
  org_name: string;
  role: string;
}

const inputClass = "w-full border border-v-border rounded-md px-3 py-2 text-sm text-v-text bg-v-bg placeholder:text-v-muted focus:outline-none focus:ring-2 focus:ring-v-accent";

export default function InviteAcceptPage() {
  const { token } = useParams<{ token: string }>();
  const router = useRouter();
  const [info, setInfo] = useState<InvitationInfo | null>(null);
  const [error, setError] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetch(`/api/invitations/${token}`)
      .then(async (r) => {
        if (!r.ok) {
          const body = await r.json().catch(() => ({}));
          setError(body.error ?? "Convite inválido ou expirado.");
          return;
        }
        return r.json();
      })
      .then((d) => d && setInfo(d));
  }, [token]);

  async function handleAccept(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    const res = await fetch(`/api/invitations/${token}/accept`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, password }),
    });
    setSubmitting(false);
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      setError(body.error ?? "Erro ao aceitar convite.");
      return;
    }
    router.push("/dashboard");
    router.refresh();
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-v-bg">
        <div className="bg-v-card rounded-xl border border-v-border p-8 max-w-sm w-full text-center">
          <p className="text-red-400 font-medium">{error}</p>
          <a href="/login" className="mt-4 inline-block text-sm text-v-accent hover:underline">Ir para o login</a>
        </div>
      </div>
    );
  }

  if (!info) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-v-bg">
        <p className="text-v-muted text-sm">Validando convite...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-v-bg">
      <div className="bg-v-card rounded-xl border border-v-border p-8 max-w-sm w-full space-y-6">
        <div className="text-center">
          <h1 className="text-xl font-bold text-v-accent">Bem-vindo ao Vantaggio Hunter</h1>
          <p className="text-sm text-v-muted mt-1">
            Você foi convidado para <span className="font-medium text-v-text/80">{info.org_name}</span> como <span className="font-medium text-v-text/80">{info.role}</span>.
          </p>
          <p className="text-xs text-v-muted mt-1">{info.email}</p>
        </div>

        <form onSubmit={handleAccept} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-v-muted mb-1">Seu nome</label>
            <input
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              className={inputClass}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-v-muted mb-1">Criar senha</label>
            <input
              type="password"
              required
              minLength={6}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={inputClass}
            />
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <button
            type="submit"
            disabled={submitting}
            className="w-full py-2 bg-v-accent text-white text-sm font-medium rounded-md hover:bg-v-glow disabled:opacity-50"
          >
            {submitting ? "Criando conta..." : "Aceitar Convite"}
          </button>
        </form>
      </div>
    </div>
  );
}
