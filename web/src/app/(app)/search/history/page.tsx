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
  queued:     { label: "Na fila",      className: "bg-yellow-50 text-yellow-700" },
  processing: { label: "Processando",  className: "bg-blue-50 text-blue-700" },
  done:       { label: "Concluída",    className: "bg-green-50 text-green-700" },
  failed:     { label: "Falhou",       className: "bg-red-50 text-red-700" },
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
          className="p-1.5 rounded-lg border border-gray-200 hover:bg-gray-50 text-gray-500"
        >
          <ArrowLeft size={16} />
        </button>
        <div>
          <h1 className="text-xl font-semibold text-gray-900 flex items-center gap-2">
            <Clock size={18} className="text-gray-400" />
            Histórico de Buscas
          </h1>
          {data && (
            <p className="text-sm text-gray-500 mt-0.5">
              {data.total.toLocaleString("pt-BR")} busca{data.total !== 1 ? "s" : ""}
            </p>
          )}
        </div>
      </div>

      <div className="flex-1 bg-white border border-gray-200 rounded-xl overflow-hidden flex flex-col min-h-0">
        {/* Header */}
        <div className="grid grid-cols-[160px_80px_1fr_120px_80px] gap-2 px-4 py-3 bg-gray-50 border-b border-gray-100 text-xs font-medium text-gray-600 shrink-0">
          <span>Data</span>
          <span>Modo</span>
          <span>Filtros / Consulta</span>
          <span>Resultados</span>
          <span>Status</span>
        </div>

        <div className="flex-1 overflow-auto">
          {loading ? (
            <div className="flex items-center justify-center h-40 text-sm text-gray-400">
              Carregando...
            </div>
          ) : searches.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-40 gap-2 text-sm text-gray-400">
              <Clock size={32} className="text-gray-200" />
              Nenhuma busca realizada ainda
            </div>
          ) : (
            searches.map((s) => {
              const st = STATUS_LABEL[s.status] ?? { label: s.status, className: "bg-gray-100 text-gray-600" };
              return (
                <button
                  key={s.id}
                  onClick={() => router.push(`/search/${s.id}`)}
                  className="w-full grid grid-cols-[160px_80px_1fr_120px_80px] gap-2 items-center px-4 py-3 border-b border-gray-50 hover:bg-indigo-50 text-left text-sm transition-colors"
                >
                  <span className="text-gray-600 text-xs">{formatDate(s.created_at)}</span>
                  <span className="text-gray-700 font-medium">{MODE_LABEL[s.mode] ?? s.mode}</span>
                  <span className="text-gray-600 truncate text-xs" title={summarizeFilters(s)}>
                    {summarizeFilters(s)}
                  </span>
                  <span className="text-gray-700 tabular-nums">
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
          <div className="flex items-center justify-between px-4 py-3 border-t border-gray-100 shrink-0">
            <span className="text-xs text-gray-500">Página {page} de {totalPages}</span>
            <div className="flex gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-3 py-1 text-xs border border-gray-300 rounded-lg disabled:opacity-40 hover:bg-gray-50"
              >
                Anterior
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                className="px-3 py-1 text-xs border border-gray-300 rounded-lg disabled:opacity-40 hover:bg-gray-50"
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
