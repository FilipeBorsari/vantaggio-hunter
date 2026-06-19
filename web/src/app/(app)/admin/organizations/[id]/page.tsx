"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";

interface OrgUser {
  user_id: string;
  name: string;
  email: string;
  role: string;
  is_active: boolean;
  searches_this_month: number;
  exports_this_month: number;
  credits_consumed: number;
  last_active_at: string | null;
}

interface OrgDetail {
  org: { id: string; name: string; plan_name?: string; is_active: boolean; created_at: string };
  stats: { balance: number; total_searches: number; exports: number };
  users: OrgUser[];
}

const modalInputClass = "w-full border border-v-border rounded-md px-3 py-2 text-sm text-v-text bg-v-bg placeholder:text-v-muted focus:outline-none focus:ring-2 focus:ring-v-accent";

export default function AdminOrgDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [data, setData] = useState<OrgDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<"users" | "credits">("users");
  const [addCreditsOpen, setAddCreditsOpen] = useState(false);
  const [creditsAmount, setCreditsAmount] = useState("");
  const [creditsDesc, setCreditsDesc] = useState("");

  useEffect(() => {
    fetch(`/api/admin/organizations/${id}`)
      .then((r) => r.json())
      .then(setData)
      .finally(() => setLoading(false));
  }, [id]);

  async function handleAddCredits() {
    const res = await fetch(`/api/admin/organizations/${id}/credits`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ amount: parseInt(creditsAmount), description: creditsDesc }),
    });
    if (res.ok) {
      setAddCreditsOpen(false);
      setCreditsAmount("");
      setCreditsDesc("");
      const updated = await fetch(`/api/admin/organizations/${id}`).then((r) => r.json());
      setData(updated);
    }
  }

  async function handleImpersonate() {
    const res = await fetch(`/api/admin/organizations/${id}/impersonate`, { method: "POST" });
    if (res.ok) {
      const { access_token } = await res.json();
      await fetch("/api/auth/impersonate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: access_token }),
      });
      router.push("/org");
      router.refresh();
    }
  }

  if (loading) return <div className="text-v-muted text-sm">Carregando...</div>;
  if (!data) return <div className="text-v-muted text-sm">Organização não encontrada.</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-v-text">{data.org.name}</h1>
          <p className="text-sm text-v-muted mt-0.5">
            {data.org.plan_name ?? "Sem plano"} &middot;{" "}
            <span className={data.org.is_active ? "text-green-400" : "text-red-400"}>
              {data.org.is_active ? "Ativo" : "Inativo"}
            </span>
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleImpersonate}
            className="px-3 py-1.5 text-sm border border-v-border text-v-text rounded-md hover:bg-v-border/40"
          >
            Acessar como Org
          </button>
          <button
            onClick={() => setAddCreditsOpen(true)}
            className="px-3 py-1.5 text-sm bg-v-accent text-white rounded-md hover:bg-v-glow"
          >
            Adicionar Créditos
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        <Stat label="Saldo" value={data.stats.balance.toLocaleString("pt-BR")} />
        <Stat label="Buscas (30d)" value={data.stats.total_searches} />
        <Stat label="Exports (30d)" value={data.stats.exports} />
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-v-border">
        {(["users", "credits"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 text-sm font-medium -mb-px border-b-2 transition-colors ${
              tab === t ? "border-v-accent text-v-accent" : "border-transparent text-v-muted hover:text-v-text"
            }`}
          >
            {t === "users" ? "Usuários" : "Créditos"}
          </button>
        ))}
      </div>

      {tab === "users" && (
        <div className="bg-v-card rounded-xl border border-v-card-border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-v-bg text-v-muted text-xs uppercase">
              <tr>
                <th className="px-4 py-3 text-left">Nome</th>
                <th className="px-4 py-3 text-left">Role</th>
                <th className="px-4 py-3 text-right">Buscas/mês</th>
                <th className="px-4 py-3 text-right">Créditos</th>
                <th className="px-4 py-3 text-center">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-v-card-border">
              {data.users.map((u) => (
                <tr key={u.user_id} className="hover:bg-v-border/30">
                  <td className="px-4 py-3">
                    <div className="font-medium text-v-text">{u.name}</div>
                    <div className="text-xs text-v-muted">{u.email}</div>
                  </td>
                  <td className="px-4 py-3 text-v-text/60">{u.role}</td>
                  <td className="px-4 py-3 text-right text-v-text/60">{u.searches_this_month}</td>
                  <td className="px-4 py-3 text-right text-v-text/60">{u.credits_consumed}</td>
                  <td className="px-4 py-3 text-center">
                    <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${u.is_active ? "bg-green-900/30 text-green-400" : "bg-v-border text-v-muted"}`}>
                      {u.is_active ? "Ativo" : "Inativo"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {tab === "credits" && (
        <div className="bg-v-card rounded-xl border border-v-card-border p-6 text-sm text-v-muted">
          Histórico de créditos disponível em breve.
        </div>
      )}

      {/* Add Credits Modal */}
      {addCreditsOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="bg-v-card border border-v-border rounded-xl shadow-lg p-6 w-full max-w-sm space-y-4">
            <h2 className="text-base font-semibold text-v-text">Adicionar Créditos</h2>
            <input
              type="number"
              placeholder="Quantidade"
              value={creditsAmount}
              onChange={(e) => setCreditsAmount(e.target.value)}
              className={modalInputClass}
            />
            <input
              type="text"
              placeholder="Descrição (ex: Recarga mensal)"
              value={creditsDesc}
              onChange={(e) => setCreditsDesc(e.target.value)}
              className={modalInputClass}
            />
            <div className="flex justify-end gap-2">
              <button onClick={() => setAddCreditsOpen(false)} className="px-3 py-1.5 text-sm text-v-muted hover:bg-v-border/40 rounded-md">
                Cancelar
              </button>
              <button onClick={handleAddCredits} className="px-3 py-1.5 text-sm bg-v-accent text-white rounded-md hover:bg-v-glow">
                Confirmar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-v-card rounded-xl border border-v-card-border p-4">
      <p className="text-xs text-v-muted mb-1">{label}</p>
      <p className="text-xl font-bold text-v-text">{value}</p>
    </div>
  );
}
