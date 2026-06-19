"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";

interface SearchSummary {
  search_id: string;
  query: string;
  results_count: number;
  credits_used: number;
  created_at: string;
}

interface UserHistory {
  user: { user_id: string; name: string; email: string };
  stats: { searches: number; exports: number; credits_consumed: number };
  searches: SearchSummary[];
  total: number;
  page: number;
}

export default function OrgUserHistoryPage() {
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<UserHistory | null>(null);
  const [period, setPeriod] = useState("30d");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/org/users/${id}/history?period=${period}`)
      .then((r) => r.json())
      .then(setData)
      .finally(() => setLoading(false));
  }, [id, period]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-v-text">{data?.user.name ?? "Vendedor"}</h1>
          <p className="text-sm text-v-muted">{data?.user.email}</p>
        </div>
        <select
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
          className="text-sm border border-v-border rounded-md px-2 py-1 bg-v-bg text-v-text focus:outline-none focus:ring-2 focus:ring-v-accent"
        >
          <option value="7d">7 dias</option>
          <option value="30d">30 dias</option>
          <option value="90d">90 dias</option>
        </select>
      </div>

      {loading ? (
        <div className="text-v-muted text-sm">Carregando...</div>
      ) : data ? (
        <>
          <div className="grid grid-cols-3 gap-4">
            <KPI label="Buscas" value={data.stats.searches} />
            <KPI label="Exports" value={data.stats.exports} />
            <KPI label="Créditos" value={data.stats.credits_consumed} />
          </div>

          <div className="bg-v-card rounded-xl border border-v-card-border overflow-hidden">
            <div className="px-4 py-3 border-b border-v-border">
              <h2 className="text-sm font-semibold text-v-text/80">Histórico de Buscas</h2>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-v-bg text-v-muted text-xs uppercase">
                <tr>
                  <th className="px-4 py-3 text-left">Query</th>
                  <th className="px-4 py-3 text-right">Resultados</th>
                  <th className="px-4 py-3 text-right">Créditos</th>
                  <th className="px-4 py-3 text-left">Data</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-v-card-border">
                {data.searches.map((s) => (
                  <tr key={s.search_id} className="hover:bg-v-border/30">
                    <td className="px-4 py-3 text-v-text/80 max-w-xs truncate">{s.query || "—"}</td>
                    <td className="px-4 py-3 text-right text-v-text/60">{s.results_count}</td>
                    <td className="px-4 py-3 text-right text-v-text/60">{s.credits_used}</td>
                    <td className="px-4 py-3 text-v-muted text-xs">{new Date(s.created_at).toLocaleString("pt-BR")}</td>
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
      <p className="text-xl font-bold text-v-text">{value.toLocaleString("pt-BR")}</p>
    </div>
  );
}
