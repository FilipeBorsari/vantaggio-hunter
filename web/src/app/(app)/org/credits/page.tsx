"use client";

import { useEffect, useState } from "react";

interface Balance {
  balance: number;
  org_id: string;
}

export default function OrgCreditsPage() {
  const [data, setData] = useState<Balance | null>(null);

  useEffect(() => {
    fetch("/api/org/credits").then((r) => r.json()).then(setData);
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-v-text">Créditos da Organização</h1>
      {data ? (
        <div className="bg-v-card rounded-xl border border-v-card-border p-6">
          <p className="text-xs text-v-muted mb-1">Saldo Atual</p>
          <p className="text-4xl font-bold text-v-text">{data.balance.toLocaleString("pt-BR")}</p>
          <p className="text-sm text-v-muted mt-2">créditos</p>
        </div>
      ) : (
        <div className="text-v-muted text-sm">Carregando...</div>
      )}
    </div>
  );
}
