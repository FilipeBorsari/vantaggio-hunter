"use client";

import { useEffect, useState } from "react";

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

export default function OrgCostsPage() {
  const [data, setData] = useState<OrgCosts | null>(null);
  const [period, setPeriod] = useState("30d");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/org/costs?period=${period}`)
      .then((r) => r.json())
      .then(setData)
      .finally(() => setLoading(false));
  }, [period]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-v-text">Custos por Vendedor</h1>
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
          <div className="bg-v-card rounded-xl border border-v-card-border p-4 flex items-center gap-4">
            <div>
              <p className="text-xs text-v-muted">Total Consumido ({data.period})</p>
              <p className="text-2xl font-bold text-v-text">{data.total_credits_consumed.toLocaleString("pt-BR")}</p>
            </div>
          </div>

          <div className="bg-v-card rounded-xl border border-v-card-border overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-v-bg text-v-muted text-xs uppercase">
                <tr>
                  <th className="px-4 py-3 text-left">Vendedor</th>
                  <th className="px-4 py-3 text-right">Buscas</th>
                  <th className="px-4 py-3 text-right">Exports</th>
                  <th className="px-4 py-3 text-right">Créditos</th>
                  <th className="px-4 py-3 text-right">% do Total</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-v-card-border">
                {data.by_seller.map((s) => {
                  const pct = data.total_credits_consumed > 0
                    ? ((s.credits_consumed / data.total_credits_consumed) * 100).toFixed(1)
                    : "0.0";
                  return (
                    <tr key={s.user_id} className="hover:bg-v-border/30">
                      <td className="px-4 py-3 font-medium text-v-text">{s.name}</td>
                      <td className="px-4 py-3 text-right text-v-text/60">{s.searches}</td>
                      <td className="px-4 py-3 text-right text-v-text/60">{s.exports}</td>
                      <td className="px-4 py-3 text-right text-v-text font-medium">{s.credits_consumed.toLocaleString("pt-BR")}</td>
                      <td className="px-4 py-3 text-right text-v-muted">{pct}%</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </>
      ) : null}
    </div>
  );
}
