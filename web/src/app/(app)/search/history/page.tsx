"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Clock, ArrowLeft } from "lucide-react";

interface Search {
  id: string;
  mode: "structured" | "semantic";
  status: "queued" | "processing" | "done" | "failed";
  result_count?: number;
  filters: Record<string, unknown>;
  query_text?: string;
  created_at: string;
}

interface SearchListResponse {
  data: Search[];
  total: number;
}

const STATUS_LABEL: Record<string, { label: string; className: string }> = {
  queued:     { label: "Na fila",      className: "bg-yellow-900/30 text-yellow-400" },
  processing: { label: "Processando",  className: "bg-blue-900/30 text-blue-400" },
  done:       { label: "Concluída",    className: "bg-green-900/30 text-green-400" },
  failed:     { label: "Falhou",       className: "bg-red-900/30 text-red-400" },
};

const MODE_LABEL: Record<string, string> = {
  structured: "Filtros",
  semantic:   "IA",
};

function summarizeFilters(search: Search): string {
  if (search.mode === "semantic") return search.query_text ?? "";
  const f = search.filters as {
    uf?: string; city?: string; cnaes?: string[]; capital_min?: number; status?: number;
  };
  const parts: string[] = [];
  if (f.cnaes?.length) parts.push(`CNAEs: ${f.cnaes.slice(0, 2).join(", ")}${f.cnaes.length > 2 ? "…" : ""}`);
  if (f.uf) parts.push(`UF: ${f.uf}`);
  if (f.city) parts.push(`Cidade: ${f.city}`);
  if (f.capital_min) parts.push(`Capital ≥ R$${f.capital_min.toLocaleString("pt-BR")}`);
  if (f.status) parts.push(`Situação: ${f.status}`);
  return parts.join(" · ") || "Sem filtros";
}

function formatDate(iso: string) {
  return new Intl.DateTimeFormat("pt-BR", {
    day: "2-digit", month: "2-digit", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  }).format(new Date(iso));
}

export default function SearchHistoryPage() {
  const router = useRouter();
  const [data, setData] = useState<SearchListResponse | null>(null);
  const [page, setPage] = useState(1);
  const limit = 20;
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/searches?page=${page}&limit=${limit}`)
      .then((r) => r.ok ? r.json() : null)
      .then((json) => { if (json) setData(json); })
      .finally(() => setLoading(false));
  }, [page]);

  const searches = data?.data ?? [];
  const totalPages = data?.total != null ? Math.ceil(data.total / limit) : 0;

  return (
    <div className="flex flex-col h-full gap-4">
      <div className="flex items-center gap-3">
        <button
          onClick={() => router.push("/search")}
          className="p-1.5 rounded-lg border border-v-border hover:bg-v-border/40 text-v-muted"
        >
          <ArrowLeft size={16} />
        </button>
        <div>
          <h1 className="text-xl font-semibold text-v-text flex items-center gap-2">
            <Clock size={18} className="text-v-muted" />
            Histórico de Buscas
          </h1>
          {data && (
            <p className="text-sm text-v-muted mt-0.5">
              {data.total.toLocaleString("pt-BR")} busca{data.total !== 1 ? "s" : ""}
            </p>
          )}
        </div>
      </div>

      <div className="flex-1 bg-v-card border border-v-card-border rounded-xl overflow-hidden flex flex-col min-h-0">
        {/* Header */}
        <div className="grid grid-cols-[160px_80px_1fr_120px_80px] gap-2 px-4 py-3 bg-v-bg border-b border-v-border text-xs font-medium text-v-muted shrink-0">
          <span>Data</span>
          <span>Modo</span>
          <span>Filtros / Consulta</span>
          <span>Resultados</span>
          <span>Status</span>
        </div>

        <div className="flex-1 overflow-auto">
          {loading ? (
            <div className="flex items-center justify-center h-40 text-sm text-v-muted">
              Carregando...
            </div>
          ) : searches.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-40 gap-2 text-sm text-v-muted">
              <Clock size={32} className="opacity-20" />
              Nenhuma busca realizada ainda
            </div>
          ) : (
            searches.map((s) => {
              const st = STATUS_LABEL[s.status] ?? { label: s.status, className: "bg-v-border text-v-muted" };
              return (
                <button
                  key={s.id}
                  onClick={() => router.push(`/search/${s.id}`)}
                  className="w-full grid grid-cols-[160px_80px_1fr_120px_80px] gap-2 items-center px-4 py-3 border-b border-v-card-border hover:bg-v-border/30 text-left text-sm transition-colors"
                >
                  <span className="text-v-muted text-xs">{formatDate(s.created_at)}</span>
                  <span className="text-v-text/80 font-medium">{MODE_LABEL[s.mode] ?? s.mode}</span>
                  <span className="text-v-text/60 truncate text-xs" title={summarizeFilters(s)}>
                    {summarizeFilters(s)}
                  </span>
                  <span className="text-v-text/80 tabular-nums">
                    {s.result_count != null ? s.result_count.toLocaleString("pt-BR") : "—"}
                  </span>
                  <span>
                    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${st.className}`}>
                      {st.label}
                    </span>
                  </span>
                </button>
              );
            })
          )}
        </div>

        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-v-border shrink-0">
            <span className="text-xs text-v-muted">Página {page} de {totalPages}</span>
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
