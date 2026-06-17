"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ChevronLeft } from "lucide-react";
import Link from "next/link";

interface Plan {
  id: string;
  name: string;
  credits: number;
  price_cents: number;
}

interface CreatedOrg {
  id: string;
  name: string;
}

export default function NewOrganizationPage() {
  const router = useRouter();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [plansLoaded, setPlansLoaded] = useState(false);
  const [createdOrg, setCreatedOrg] = useState<CreatedOrg | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function loadPlans() {
    if (plansLoaded) return;
    try {
      const res = await fetch("/api/admin/plans");
      if (res.ok) setPlans(await res.json());
    } finally {
      setPlansLoaded(true);
    }
  }

  async function handleCreateOrg(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError("");
    setLoading(true);
    const fd = new FormData(e.currentTarget);
    const body: Record<string, unknown> = { name: fd.get("name") };
    if (fd.get("plan_id")) body.plan_id = fd.get("plan_id");

    try {
      const res = await fetch("/api/admin/organizations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        setError(err?.error ?? "Erro ao criar organização");
        return;
      }
      const org = await res.json();
      setCreatedOrg(org);
    } catch {
      setError("Erro ao conectar com o servidor");
    } finally {
      setLoading(false);
    }
  }

  async function handleCreateUser(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!createdOrg) return;
    setError("");
    setLoading(true);
    const fd = new FormData(e.currentTarget);

    try {
      const res = await fetch(`/api/admin/organizations/${createdOrg.id}/users`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: fd.get("email"),
          password: fd.get("password"),
          role: fd.get("role"),
        }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        setError(err?.error ?? "Erro ao criar usuário");
        return;
      }
      router.push("/admin/organizations");
    } catch {
      setError("Erro ao conectar com o servidor");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="max-w-lg">
      <div className="flex items-center gap-2 mb-6">
        <Link
          href="/admin/organizations"
          className="text-gray-500 hover:text-gray-700"
        >
          <ChevronLeft size={20} />
        </Link>
        <h1 className="text-xl font-semibold text-gray-900">Nova Organização</h1>
      </div>

      {error && (
        <p className="mb-4 text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
          {error}
        </p>
      )}

      {!createdOrg ? (
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h2 className="font-medium text-gray-900 mb-4">Dados da organização</h2>
          <form onSubmit={handleCreateOrg} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Nome
              </label>
              <input
                name="name"
                required
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                placeholder="Ex: Empresa XYZ"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Plano
              </label>
              <select
                name="plan_id"
                onFocus={loadPlans}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="">Sem plano</option>
                {plans.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} — {p.credits.toLocaleString()} créditos
                  </option>
                ))}
              </select>
            </div>
            <button
              type="submit"
              disabled={loading}
              className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white font-medium py-2 px-4 rounded-lg text-sm transition-colors"
            >
              {loading ? "Criando..." : "Criar Organização"}
            </button>
          </form>
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg">
            <p className="text-sm text-green-700 font-medium">
              Organização &quot;{createdOrg.name}&quot; criada com sucesso!
            </p>
          </div>
          <h2 className="font-medium text-gray-900 mb-4">Criar primeiro usuário</h2>
          <form onSubmit={handleCreateUser} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                E-mail
              </label>
              <input
                name="email"
                type="email"
                required
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                placeholder="usuario@empresa.com"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Senha
              </label>
              <input
                name="password"
                type="password"
                required
                minLength={8}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                placeholder="Mínimo 8 caracteres"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Role
              </label>
              <select
                name="role"
                required
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="manager">Manager</option>
                <option value="operator">Operator</option>
              </select>
            </div>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => router.push("/admin/organizations")}
                className="flex-1 border border-gray-300 text-gray-700 font-medium py-2 px-4 rounded-lg text-sm hover:bg-gray-50 transition-colors"
              >
                Pular
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex-1 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white font-medium py-2 px-4 rounded-lg text-sm transition-colors"
              >
                {loading ? "Criando..." : "Criar Usuário"}
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
}
