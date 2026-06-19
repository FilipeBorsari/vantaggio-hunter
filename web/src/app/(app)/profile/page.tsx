"use client";

import { useEffect, useState } from "react";

interface Profile {
  user_id: string;
  name: string;
  email: string;
  org_name: string;
  credits_consumed_this_month: number;
  credit_limit: number | null;
}

const inputClass = "w-full border border-v-border rounded-md px-3 py-2 text-sm text-v-text bg-v-bg focus:outline-none focus:ring-2 focus:ring-v-accent";

export default function ProfilePage() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [name, setName] = useState("");
  const [currentPwd, setCurrentPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    fetch("/api/me/profile")
      .then((r) => r.json())
      .then((d) => { setProfile(d); setName(d.name); });
  }, []);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSuccess(false);
    const res = await fetch("/api/me/profile", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, current_password: currentPwd, new_password: newPwd }),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      setError(body.error ?? "Erro ao salvar.");
      return;
    }
    setSuccess(true);
    setCurrentPwd("");
    setNewPwd("");
  }

  if (!profile) return <div className="text-v-muted text-sm">Carregando...</div>;

  return (
    <div className="max-w-md space-y-6">
      <h1 className="text-2xl font-bold text-v-text">Perfil</h1>

      <div className="bg-v-card rounded-xl border border-v-card-border p-4 space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-v-muted">Organização</span>
          <span className="font-medium text-v-text">{profile.org_name}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-v-muted">E-mail</span>
          <span className="text-v-text/80">{profile.email}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-v-muted">Créditos (30d)</span>
          <span className="text-v-text/80">{profile.credits_consumed_this_month}</span>
        </div>
        {profile.credit_limit !== null && (
          <div className="flex justify-between">
            <span className="text-v-muted">Limite de Créditos</span>
            <span className="text-v-text/80">{profile.credit_limit}</span>
          </div>
        )}
      </div>

      <form onSubmit={handleSave} className="bg-v-card rounded-xl border border-v-card-border p-6 space-y-4">
        <h2 className="text-sm font-semibold text-v-text/80">Atualizar Dados</h2>
        <div>
          <label className="block text-xs font-medium text-v-muted mb-1">Nome</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={inputClass}
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-v-muted mb-1">Senha Atual</label>
          <input
            type="password"
            value={currentPwd}
            onChange={(e) => setCurrentPwd(e.target.value)}
            className={inputClass}
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-v-muted mb-1">Nova Senha</label>
          <input
            type="password"
            value={newPwd}
            onChange={(e) => setNewPwd(e.target.value)}
            className={inputClass}
          />
        </div>
        {error && <p className="text-xs text-red-400">{error}</p>}
        {success && <p className="text-xs text-green-400">Salvo com sucesso.</p>}
        <button
          type="submit"
          className="px-4 py-2 bg-v-accent text-white text-sm font-medium rounded-md hover:bg-v-glow"
        >
          Salvar
        </button>
      </form>
    </div>
  );
}
