"use client";

import { useEffect, useState } from "react";
import { CreditCard, TrendingDown, TrendingUp } from "lucide-react";

interface CreditTransaction {
  id: string;
  type: string;
  amount: number;
  description?: string;
  reference_id?: string;
  created_at: string;
}

interface TransactionsResponse {
  data: CreditTransaction[];
  total: number;
  balance: number;
}

const TYPE_LABEL: Record<string, string> = {
  purchase:       "Compra",
  search:         "Busca",
  company_detail: "Consulta CNPJ",
  enrichment:     "Enriquecimento",
  export:         "Exportação",
  adjustment:     "Ajuste",
};

function formatDate(iso: string) {
  return new Intl.DateTimeFormat("pt-BR", {
    day: "2-digit", month: "2-digit", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  }).format(new Date(iso));
}

export default function CreditsPage() {
  const [data, setData] = useState<TransactionsResponse | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const limit = 20;

  useEffect(() => {
    setLoading(true);
    fetch(`/api/credits/transactions?page=${page}&limit=${limit}`)
      .then((r) => r.ok ? r.json() : null)
      .then((json) => { if (json) setData(json); })
      .finally(() => setLoading(false));
  }, [page]);

  const transactions = data?.data ?? [];
  const totalPages = data?.total != null ? Math.ceil(data.total / limit) : 0;
  const balance = data?.balance ?? null;

  const balanceColor =
    balance === 0
      ? "text-red-400"
      : balance !== null && balance < 100
      ? "text-amber-400"
      : "text-v-text";

  return (
    <div className="max-w-3xl flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold text-v-text">Créditos</h1>
        <p className="text-sm text-v-muted mt-0.5">Saldo atual e histórico de consumo da sua organização.</p>
      </div>

      {/* Balance card */}
      <div className="bg-v-card border border-v-card-border rounded-xl p-5 flex items-center gap-4">
        <div className="p-3 bg-v-accent/10 rounded-xl">
          <CreditCard size={24} className="text-v-accent" />
        </div>
        <div>
          <p className="text-xs text-v-muted uppercase tracking-wide font-medium">Saldo disponível</p>
          <p className={`text-3xl font-bold tabular-nums ${balanceColor}`}>
            {balance !== null ? balance.toLocaleString("pt-BR") : "—"}
          </p>
        </div>
        {balance !== null && balance < 100 && (
          <div className="ml-auto text-sm text-amber-400 bg-amber-900/30 border border-amber-800 rounded-lg px-3 py-2">
            Saldo baixo. Entre em contato com o administrador.
          </div>
        )}
      </div>

      {/* Transactions */}
      <div className="bg-v-card border border-v-card-border rounded-xl overflow-hidden flex flex-col">
        <div className="grid grid-cols-[160px_100px_1fr_100px] gap-2 px-4 py-3 bg-v-bg border-b border-v-border text-xs font-medium text-v-muted">
          <span>Data</span>
          <span>Tipo</span>
          <span>Descrição</span>
          <span className="text-right">Créditos</span>
        </div>

        <div className="flex-1 overflow-auto">
          {loading ? (
            <div className="flex items-center justify-center h-32 text-sm text-v-muted">
              Carregando...
            </div>
          ) : transactions.length === 0 ? (
            <div className="flex items-center justify-center h-32 text-sm text-v-muted">
              Nenhuma transação ainda
            </div>
          ) : (
            transactions.map((tx) => (
              <div
                key={tx.id}
                className="grid grid-cols-[160px_100px_1fr_100px] gap-2 items-center px-4 py-3 border-b border-v-card-border text-sm"
              >
                <span className="text-v-muted text-xs">{formatDate(tx.created_at)}</span>
                <span className="text-v-text/80">{TYPE_LABEL[tx.type] ?? tx.type}</span>
                <span className="text-v-text/60 truncate text-xs" title={tx.description}>
                  {tx.description ?? "—"}
                </span>
                <span className={`text-right font-semibold tabular-nums flex items-center justify-end gap-1 ${
                  tx.amount > 0 ? "text-green-400" : "text-red-400"
                }`}>
                  {tx.amount > 0 ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
                  {tx.amount > 0 ? "+" : ""}{tx.amount.toLocaleString("pt-BR")}
                </span>
              </div>
            ))
          )}
        </div>

        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-v-border shrink-0">
            <span className="text-xs text-v-muted">
              Página {page} de {totalPages} — {(data?.total ?? 0).toLocaleString("pt-BR")} transações
            </span>
            <div className="flex gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-3 py-1 text-xs border border-v-border text-v-muted rounded-lg disabled:opacity-40 hover:bg-v-border/40 hover:text-v-text"
              >
                Anterior
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                className="px-3 py-1 text-xs border border-v-border text-v-muted rounded-lg disabled:opacity-40 hover:bg-v-border/40 hover:text-v-text"
              >
                Próxima
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
