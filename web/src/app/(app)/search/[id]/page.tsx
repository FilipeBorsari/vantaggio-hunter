"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowLeft, RefreshCw } from "lucide-react";

interface SearchResult {
  cnpj: string;
  razao_social: string;
  municipio?: string;
  uf: string;
  capital_social?: number;
  situacao: number;
  score?: number;
}

interface SearchResponse {
  id: string;
  mode?: "structured" | "semantic";
  status: "queued" | "processing" | "done" | "failed";
  result_count?: number;
  results?: SearchResult[];
  page?: number;
  limit?: number;
  total?: number;
}

const SITUACAO_LABEL: Record<number, string> = {
  1: "Nula", 2: "Ativa", 3: "Suspensa", 4: "Inapta", 8: "Baixada",
};

function formatCNPJ(cnpj: string) {
  return cnpj.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, "$1.$2.$3/$4-$5");
}

function formatCurrency(v?: number) {
  if (v == null) return "—";
  return new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL", maximumFractionDigits: 0 }).format(v);
}

function SkeletonRow({ isSemantic }: { isSemantic: boolean }) {
  const cols = isSemantic ? [200, 300, 120, 80, 60, 50] : [200, 300, 120, 80, 60];
  const grid = isSemantic
    ? "grid-cols-[200px_1fr_160px_120px_80px_64px]"
    : "grid-cols-[200px_1fr_160px_120px_80px]";
  return (
    <div className={`grid ${grid} gap-2 items-center px-4 h-12 border-b border-gray-50 animate-pulse`}>
      {cols.map((w, i) => (
        <div key={i} className="h-3 bg-gray-200 rounded" style={{ width: `${Math.min(w, 100)}%` }} />
      ))}
    </div>
  );
}

export default function SearchResultsPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [data, setData] = useState<SearchResponse | null>(null);
  const [page, setPage] = useState(1);
  const limit = 100;
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollStart = useRef(Date.now());

  const fetchResults = useCallback(async (p: number) => {
    const res = await fetch(`/api/searches/${id}?page=${p}&limit=${limit}`);
    if (!res.ok) return;
    const json: SearchResponse = await res.json();
    setData(json);
    return json;
  }, [id]);

  useEffect(() => {
    fetchResults(page);

    pollRef.current = setInterval(async () => {
      if (Date.now() - pollStart.current > 60_000) {
        clearInterval(pollRef.current!);
        return;
      }
      const json = await fetchResults(page);
      if (json?.status === "done" || json?.status === "failed") {
        clearInterval(pollRef.current!);
      }
    }, 2000);

    return () => { if (pollRef.current) clearInterval(pollRef.current); };
  }, [fetchResults, page]);

  const results = data?.results ?? [];
  const isSemantic = data?.mode === "semantic";
  const gridCols = isSemantic
    ? "grid-cols-[200px_1fr_160px_120px_80px_64px]"
    : "grid-cols-[200px_1fr_160px_120px_80px]";
  const parentRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line react-hooks/incompatible-library -- useVirtualizer não é memoizável por design do TanStack
  const rowVirtualizer = useVirtualizer({
    count: results.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 48,
    overscan: 10,
  });

  const totalPages = data?.total != null ? Math.ceil(data.total / limit) : 0;
  const isLoading = !data || data.status === "queued" || data.status === "processing";

  async function handlePageChange(p: number) {
    setPage(p);
    const res = await fetch(`/api/searches/${id}?page=${p}&limit=${limit}`);
    if (res.ok) setData(await res.json());
  }

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
          <h1 className="text-xl font-semibold text-gray-900">
            {isLoading ? "Processando busca..." : data?.status === "failed" ? "Busca falhou" : `${(data?.result_count ?? 0).toLocaleString("pt-BR")} empresas encontradas`}
          </h1>
          <p className="text-xs text-gray-500 font-mono mt-0.5">{id}</p>
        </div>
        {isLoading && (
          <RefreshCw size={16} className="ml-2 text-indigo-500 animate-spin" />
        )}
      </div>

      {data?.status === "failed" && (
        <div className="bg-red-50 border border-red-200 rounded-xl px-4 py-3 text-sm text-red-700">
          A busca falhou. Tente criar uma nova busca.
        </div>
      )}

      <div className="flex-1 bg-white border border-gray-200 rounded-xl overflow-hidden flex flex-col min-h-0">
        {/* Header */}
        <div className={`grid ${gridCols} gap-2 px-4 py-3 bg-gray-50 border-b border-gray-100 text-xs font-medium text-gray-600 shrink-0`}>
          <span>CNPJ</span>
          <span>Razão Social</span>
          <span>Município/UF</span>
          <span>Capital</span>
          <span>Situação</span>
          {isSemantic && <span>Score</span>}
        </div>

        {/* Body */}
        <div ref={parentRef} className="flex-1 overflow-auto">
          {isLoading ? (
            <div className="flex flex-col">
              {Array.from({ length: 8 }).map((_, i) => <SkeletonRow key={i} isSemantic={isSemantic} />)}
              <div className="flex items-center justify-center py-6 text-sm text-gray-400 gap-2">
                <RefreshCw size={14} className="animate-spin" />
                Processando sua busca...
              </div>
            </div>
          ) : results.length === 0 ? (
            <div className="flex items-center justify-center h-40 text-sm text-gray-400">
              Nenhum resultado encontrado
            </div>
          ) : (
            <div style={{ height: rowVirtualizer.getTotalSize(), position: "relative" }}>
              {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                const r = results[virtualRow.index];
                return (
                  <div
                    key={virtualRow.key}
                    style={{ position: "absolute", top: virtualRow.start, width: "100%", height: virtualRow.size }}
                    className={`grid ${gridCols} gap-2 items-center px-4 border-b border-gray-50 hover:bg-gray-50 text-sm`}
                  >
                    <span className="font-mono text-xs text-gray-700 truncate">{formatCNPJ(r.cnpj)}</span>
                    <span className="font-medium text-gray-900 truncate" title={r.razao_social}>{r.razao_social}</span>
                    <span className="text-gray-600 truncate">{[r.municipio, r.uf].filter(Boolean).join(" / ")}</span>
                    <span className="text-gray-600 tabular-nums">{formatCurrency(r.capital_social)}</span>
                    <span>
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                        r.situacao === 2 ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-600"
                      }`}>
                        {SITUACAO_LABEL[r.situacao] ?? r.situacao}
                      </span>
                    </span>
                    {isSemantic && (
                      <span className="text-gray-600 tabular-nums text-xs">
                        {r.score != null ? Math.round(r.score * 100) : "—"}
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-gray-100 shrink-0">
            <span className="text-xs text-gray-500">
              Página {page} de {totalPages} — {(data?.total ?? 0).toLocaleString("pt-BR")} resultados
            </span>
            <div className="flex gap-2">
              <button
                onClick={() => handlePageChange(Math.max(1, page - 1))}
                disabled={page === 1}
                className="px-3 py-1 text-xs border border-gray-300 rounded-lg disabled:opacity-40 hover:bg-gray-50"
              >
                Anterior
              </button>
              <button
                onClick={() => handlePageChange(Math.min(totalPages, page + 1))}
                disabled={page === totalPages}
                className="px-3 py-1 text-xs border border-gray-300 rounded-lg disabled:opacity-40 hover:bg-gray-50"
              >
                Próxima
              </button>
            </div>
          </div>
        )}
      </div>

      {data?.status === "done" && (
        <div className="flex justify-end">
          <a
            href="/search"
            className="inline-flex items-center gap-2 text-sm text-indigo-600 hover:text-indigo-800 font-medium"
          >
            Nova Busca →
          </a>
        </div>
      )}
    </div>
  );
}
