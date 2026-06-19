"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

interface SellerCost {
  user_id: string;
  name: string;
  searches: number;
  exports: number;
  credits_consumed: number;
}

interface OrgCosts {
  period: string;
  total_credits_consumed: number;
  by_seller: SellerCost[];
}

const selectClass = "text-sm border border-v-border rounded-md px-2 py-1 bg-v-bg text-v-text focus:outline-none focus:ring-2 focus:ring-v-accent";

export default function OrgDashboardPage() {
  const [costs, setCosts] = useState<OrgCosts | null>(null);
  const [loading, setLoading] = useState(true);
  const [sellerFilter, setSellerFilter] = useState("all");
  const [period, setPeriod] = useState("30d");

  useEffect(() => {
    setLoading(true);
    fetch(`/api/org/costs?period=${period}`)
      .then((r) => r.json())
      .then(setCosts)
      .finally(() => setLoading(false));
  }, [period]);

  const sellers = costs?.by_seller ?? [];
  const displayed = sellerFilter === "all" ? sellers : sellers.filter((s) => s.user_id === sellerFilter);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-v-text">Dashboard da Organização</h1>
        <div className="flex gap-2">
          <select
            value={sellerFilter}
            onChange={(e) => setSellerFilter(e.target.value)}
            className={selectClass}
          >
            <option value="all">Todos os Vendedores</option>
            {sellers.map((s) => (
              <option key={s.user_id} value={s.user_id}>{s.name}</option>
            ))}
          </select>
          <select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className={selectClass}
          >
            <option value="7d">7 dias</option>
            <option value="30d">30 dias</option>
            <option value="90d">90 dias</option>
          </select>
        </div>
      </div>

      {loading ? (
        <div className="text-v-muted text-sm">Carregando...</div>
      ) : costs ? (
        <>
          <div className="grid grid-cols-3 gap-4">
            <KPI label="Créditos Consumidos" value={costs.total_credits_consumed.toLocaleString("pt-BR")} />
            <KPI label="Vendedores" value={sellers.length} />
            <KPI label="Total Buscas" value={sellers.reduce((a, s) => a + s.searches, 0)} />
          </div>

          <div className="bg-v-card rounded-xl border border-v-card-border overflow-hidden">
            <div className="px-4 py-3 border-b border-v-border flex items-center justify-between">
              <h2 className="text-sm font-semibold text-v-text/80">Vendedores</h2>
              <Link href="/org/users" className="text-xs text-v-accent hover:underline">Gerenciar</Link>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-v-bg text-v-muted text-xs uppercase">
                <tr>
                  <th className="px-4 py-3 text-left">Nome</th>
                  <th className="px-4 py-3 text-right">Buscas</th>
                  <th className="px-4 py-3 text-right">Exports</th>
                  <th className="px-4 py-3 text-right">Créditos</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-v-card-border">
                {displayed.map((s) => (
                  <tr key={s.user_id} className="hover:bg-v-border/30">
                    <td className="px-4 py-3 font-medium text-v-text">{s.name}</td>
                    <td className="px-4 py-3 text-right text-v-text/60">{s.searches}</td>
                    <td className="px-4 py-3 text-right text-v-text/60">{s.exports}</td>
                    <td className="px-4 py-3 text-right text-v-text/60">{s.credits_consumed}</td>
                    <td className="px-4 py-3 text-right">
                      <Link href={`/org/users/${s.user_id}`} className="text-xs text-v-accent hover:underline">
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

function KPI({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-v-card rounded-xl border border-v-card-border p-4">
      <p className="text-xs text-v-muted mb-1">{label}</p>
      <p className="text-xl font-bold text-v-text">{value}</p>
    </div>
  );
}
