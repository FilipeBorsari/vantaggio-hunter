"use client";

import { useEffect, useState } from "react";

interface SearchSummary {
  search_id: string;
  query: string;
  results_count: number;
  credits_used: number;
  created_at: string;
}

interface SearchesResponse {
  data: SearchSummary[];
  total: number;
  page: number;
}

export default function MySearchesPage() {
  const [data, setData] = useState<SearchesResponse | null>(null);
  const [period, setPeriod] = useState("30d");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/me/searches?period=${period}`)
      .then((r) => r.json())
      .then(setData)
      .finally(() => setLoading(false));
  }, [period]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-v-text">Minhas Pesquisas</h1>
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
        <div className="bg-v-card rounded-xl border border-v-card-border overflow-hidden">
          <div className="px-4 py-3 border-b border-v-border">
            <span className="text-sm text-v-muted">{data.total} pesquisas no período</span>
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
              {data.data.map((s) => (
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
      ) : null}
    </div>
  );
}
