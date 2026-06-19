"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

interface OrgSummary {
  org_id: string;
  name: string;
  searches: number;
  exports: number;
  balance: number;
  is_active: boolean;
}

interface Dashboard {
  total_orgs: number;
  active_orgs: number;
  total_searches: number;
  total_exports: number;
  total_credits_consumed: number;
  orgs: OrgSummary[];
}

export default function AdminDashboardPage() {
  const [data, setData] = useState<Dashboard | null>(null);
  const [period, setPeriod] = useState("30d");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/admin/dashboard?period=${period}`)
      .then((r) => r.json())
      .then(setData)
      .finally(() => setLoading(false));
  }, [period]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-v-text">Dashboard Global</h1>
        <div className="flex items-center gap-2">
          <select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className="text-sm border border-v-border rounded-md px-2 py-1 bg-v-bg text-v-text focus:outline-none focus:ring-2 focus:ring-v-accent"
          >
            <option value="7d">7 dias</option>
            <option value="30d">30 dias</option>
            <option value="90d">90 dias</option>
          </select>
          <Link
            href="/admin/organizations/new"
            className="px-3 py-1.5 bg-v-accent text-white text-sm font-medium rounded-md hover:bg-v-glow"
          >
            Nova Org
          </Link>
        </div>
      </div>

      {loading ? (
        <div className="text-v-muted text-sm">Carregando...</div>
      ) : data ? (
        <>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <KPI label="Total de Orgs" value={data.total_orgs} />
            <KPI label="Orgs Ativas" value={data.active_orgs} />
            <KPI label="Buscas" value={data.total_searches} />
            <KPI label="Exports" value={data.total_exports} />
            <KPI label="Créditos Consumidos" value={data.total_credits_consumed} />
          </div>

          <div className="bg-v-card rounded-xl border border-v-card-border overflow-hidden">
            <div className="px-4 py-3 border-b border-v-border">
              <h2 className="text-sm font-semibold text-v-text/80">Organizações</h2>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-v-bg text-v-muted text-xs uppercase">
                <tr>
                  <th className="px-4 py-3 text-left">Nome</th>
                  <th className="px-4 py-3 text-right">Buscas</th>
                  <th className="px-4 py-3 text-right">Exports</th>
                  <th className="px-4 py-3 text-right">Saldo</th>
                  <th className="px-4 py-3 text-center">Status</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-v-card-border">
                {data.orgs.map((org) => (
                  <tr key={org.org_id} className="hover:bg-v-border/30">
                    <td className="px-4 py-3 font-medium text-v-text">{org.name}</td>
                    <td className="px-4 py-3 text-right text-v-text/60">{org.searches}</td>
                    <td className="px-4 py-3 text-right text-v-text/60">{org.exports}</td>
                    <td className="px-4 py-3 text-right text-v-text/60">{org.balance.toLocaleString("pt-BR")}</td>
                    <td className="px-4 py-3 text-center">
                      <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${org.is_active ? "bg-green-900/30 text-green-400" : "bg-v-border text-v-muted"}`}>
                        {org.is_active ? "Ativo" : "Inativo"}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Link href={`/admin/organizations/${org.org_id}`} className="text-v-accent hover:underline text-xs">
                        Ver
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      ) : null}
    </div>
  );
}

function KPI({ label, value }: { label: string; value: number }) {
  return (
    <div className="bg-v-card rounded-xl border border-v-card-border p-4">
      <p className="text-xs text-v-muted mb-1">{label}</p>
      <p className="text-2xl font-bold text-v-text">{value.toLocaleString("pt-BR")}</p>
    </div>
  );
}
