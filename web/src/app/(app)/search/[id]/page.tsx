"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ArrowLeft, RefreshCw, Sparkles, Upload, X } from "lucide-react";

interface SearchResult {
  cnpj: string;
  razao_social: string;
  municipio?: string;
  uf: string;
  capital_social?: number;
  situacao: number;
  score?: number;
  ai_score?: number;
  ai_score_age_days?: number;
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
  const cols = isSemantic ? [200, 300, 120, 80, 60, 50, 80] : [200, 300, 120, 80, 60, 80];
  const grid = isSemantic
    ? "grid-cols-[28px_200px_1fr_160px_120px_80px_64px_96px]"
    : "grid-cols-[28px_200px_1fr_160px_120px_80px_96px]";
  return (
    <div className={`grid ${grid} gap-2 items-center px-4 h-12 border-b border-gray-50 animate-pulse`}>
      <div className="h-4 w-4 bg-gray-200 rounded" />
      {cols.map((w, i) => (
        <div key={i} className="h-3 bg-gray-200 rounded" style={{ width: `${Math.min(w, 100)}%` }} />
      ))}
    </div>
  );
}

interface ExportModalProps {
  count: number;
  searchId: string;
  selectedCNPJs: string[];
  onClose: () => void;
  onSuccess: () => void;
}

function ExportModal({ count, searchId, selectedCNPJs, onClose, onSuccess }: ExportModalProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleConfirm() {
    setError(null);
    setLoading(true);
    try {
      const res = await fetch("/api/exports", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ cnpjs: selectedCNPJs, search_id: searchId }),
      });
      const data = await res.json();
      if (!res.ok) {
        if (res.status === 404) {
          setError("Nenhuma integração CRM configurada. Configure em Configurações → CRM.");
        } else if (res.status === 402) {
          setError("Créditos insuficientes para esta exportação.");
        } else {
          setError(data.error ?? "Erro ao iniciar exportação");
        }
        return;
      }
      onSuccess();
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />
      <div className="relative bg-white rounded-2xl shadow-xl p-6 w-full max-w-sm mx-4 flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold text-gray-900">Exportar para CRM</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X size={18} />
          </button>
        </div>
        <div className="bg-gray-50 rounded-xl px-4 py-3 text-sm text-gray-700 space-y-1">
          <div className="flex justify-between">
            <span>Leads selecionados</span>
            <span className="font-medium">{count}</span>
          </div>
          <div className="flex justify-between">
            <span>Custo total</span>
            <span className="font-medium text-indigo-700">{count} crédito{count !== 1 ? "s" : ""}</span>
          </div>
        </div>
        <p className="text-xs text-gray-500">
          O débito ocorre por lead enviado com sucesso. Leads que falharem não serão cobrados.
        </p>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <div className="flex gap-2">
          <button
            onClick={onClose}
            className="flex-1 px-4 py-2 border border-gray-300 text-sm rounded-lg hover:bg-gray-50"
          >
            Cancelar
          </button>
          <button
            onClick={handleConfirm}
            disabled={loading}
            className="flex-1 px-4 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700 disabled:opacity-50"
          >
            {loading ? "Enviando..." : "Confirmar export"}
          </button>
        </div>
      </div>
    </div>
  );
}

function AIScoreBadge({ score }: { score: number }) {
  const color =
    score > 70 ? "bg-green-50 text-green-700 border-green-200"
    : score >= 40 ? "bg-yellow-50 text-yellow-700 border-yellow-200"
    : "bg-red-50 text-red-700 border-red-200";
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold border ${color}`}>
      {score}
    </span>
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

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [showModal, setShowModal] = useState(false);
  const [exportToast, setExportToast] = useState(false);

  // AI qualification state
  const [qualifying, setQualifying] = useState<Set<string>>(new Set());
  const [aiScores, setAiScores] = useState<Map<string, { score: number; age_days: number }>>(new Map());
  const [showQualifyModal, setShowQualifyModal] = useState(false);
  const [qualifyBatch, setQualifyBatch] = useState<string[]>([]);
  const [qualifyBatchLoading, setQualifyBatchLoading] = useState(false);

  async function handleQualify(cnpj: string) {
    setQualifying((prev) => new Set(prev).add(cnpj));
    try {
      const res = await fetch(`/api/ia/qualify/${cnpj}`, { method: "POST" });
      if (res.ok) {
        const json = await res.json();
        setAiScores((prev) => new Map(prev).set(cnpj, { score: json.score, age_days: 0 }));
      }
    } finally {
      setQualifying((prev) => { const next = new Set(prev); next.delete(cnpj); return next; });
    }
  }

  async function handleQualifyBatch() {
    setQualifyBatchLoading(true);
    try {
      await Promise.allSettled(
        qualifyBatch.map(async (cnpj) => {
          const res = await fetch(`/api/ia/qualify/${cnpj}`, { method: "POST" });
          if (res.ok) {
            const json = await res.json();
            setAiScores((prev) => new Map(prev).set(cnpj, { score: json.score, age_days: 0 }));
          }
        })
      );
    } finally {
      setQualifyBatchLoading(false);
      setShowQualifyModal(false);
      setQualifyBatch([]);
    }
  }

  const fetchResults = useCallback(async (p: number) => {
    const res = await fetch(`/api/searches/${id}?page=${p}&limit=${limit}`);
    if (!res.ok) return;
    const json: SearchResponse = await res.json();
    setData(json);
    if (json.results) {
      setAiScores((prev) => {
        const next = new Map(prev);
        for (const r of json.results!) {
          if (r.ai_score != null) {
            next.set(r.cnpj, { score: r.ai_score, age_days: r.ai_score_age_days ?? 0 });
          }
        }
        return next;
      });
    }
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
    ? "grid-cols-[28px_200px_1fr_160px_120px_80px_64px_96px]"
    : "grid-cols-[28px_200px_1fr_160px_120px_80px_96px]";
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

  const allPageSelected = results.length > 0 && results.every((r) => selected.has(r.cnpj));

  function toggleAll() {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allPageSelected) {
        results.forEach((r) => next.delete(r.cnpj));
      } else {
        results.forEach((r) => next.add(r.cnpj));
      }
      return next;
    });
  }

  function toggleOne(cnpj: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(cnpj) ? next.delete(cnpj) : next.add(cnpj);
      return next;
    });
  }

  async function handlePageChange(p: number) {
    setPage(p);
    const res = await fetch(`/api/searches/${id}?page=${p}&limit=${limit}`);
    if (res.ok) setData(await res.json());
  }

  function handleExportSuccess() {
    setShowModal(false);
    setSelected(new Set());
    setExportToast(true);
    setTimeout(() => setExportToast(false), 4000);
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

      {/* Export/qualify action bar */}
      {selected.size > 0 && (
        <div className="flex items-center justify-between bg-indigo-50 border border-indigo-200 rounded-xl px-4 py-2.5">
          <span className="text-sm text-indigo-700 font-medium">
            {selected.size} lead{selected.size !== 1 ? "s" : ""} selecionado{selected.size !== 1 ? "s" : ""}
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setSelected(new Set())}
              className="text-xs text-indigo-500 hover:text-indigo-700"
            >
              Limpar seleção
            </button>
            <button
              onClick={() => {
                const unqualified = Array.from(selected).filter((c) => !aiScores.has(c));
                setQualifyBatch(unqualified);
                setShowQualifyModal(true);
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-purple-600 text-white text-xs font-medium rounded-lg hover:bg-purple-700"
            >
              <Sparkles size={12} />
              Qualificar ({Array.from(selected).filter((c) => !aiScores.has(c)).length * 10} créditos)
            </button>
            <button
              onClick={() => setShowModal(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600 text-white text-xs font-medium rounded-lg hover:bg-indigo-700"
            >
              <Upload size={12} />
              Exportar para CRM ({selected.size})
            </button>
          </div>
        </div>
      )}

      {exportToast && (
        <div className="flex items-center gap-2 bg-green-50 border border-green-200 rounded-xl px-4 py-3 text-sm text-green-700">
          Export iniciado com sucesso — acompanhe em{" "}
          <a href="/exports" className="underline font-medium">Exportações</a>
        </div>
      )}

      <div className="flex-1 bg-white border border-gray-200 rounded-xl overflow-hidden flex flex-col min-h-0">
        {/* Header */}
        <div className={`grid ${gridCols} gap-2 px-4 py-3 bg-gray-50 border-b border-gray-100 text-xs font-medium text-gray-600 shrink-0`}>
          <input
            type="checkbox"
            checked={allPageSelected && results.length > 0}
            onChange={toggleAll}
            disabled={isLoading || results.length === 0}
            className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
          />
          <span>CNPJ</span>
          <span>Razão Social</span>
          <span>Município/UF</span>
          <span>Capital</span>
          <span>Situação</span>
          {isSemantic && <span>Score</span>}
          <span className="flex items-center gap-1"><Sparkles size={11} className="text-purple-500" />Score IA</span>
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
                const isChecked = selected.has(r.cnpj);
                return (
                  <div
                    key={virtualRow.key}
                    style={{ position: "absolute", top: virtualRow.start, width: "100%", height: virtualRow.size }}
                    className={`grid ${gridCols} gap-2 items-center px-4 border-b border-gray-50 hover:bg-gray-50 text-sm ${isChecked ? "bg-indigo-50/40" : ""}`}
                  >
                    <input
                      type="checkbox"
                      checked={isChecked}
                      onChange={() => toggleOne(r.cnpj)}
                      className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                    />
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
                    <span>
                      {(() => {
                        const q = aiScores.get(r.cnpj);
                        if (q) return <AIScoreBadge score={q.score} />;
                        const isQualifying = qualifying.has(r.cnpj);
                        return (
                          <button
                            onClick={() => handleQualify(r.cnpj)}
                            disabled={isQualifying}
                            className="inline-flex items-center gap-1 px-2 py-0.5 text-xs text-purple-600 border border-purple-200 rounded-full hover:bg-purple-50 disabled:opacity-50 disabled:cursor-wait"
                          >
                            <Sparkles size={10} />
                            {isQualifying ? "..." : "10cr"}
                          </button>
                        );
                      })()}
                    </span>
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

      {showModal && (
        <ExportModal
          count={selected.size}
          searchId={id}
          selectedCNPJs={Array.from(selected)}
          onClose={() => setShowModal(false)}
          onSuccess={handleExportSuccess}
        />
      )}

      {showQualifyModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40" onClick={() => setShowQualifyModal(false)} />
          <div className="relative bg-white rounded-2xl shadow-xl p-6 w-full max-w-sm mx-4 flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold text-gray-900">Qualificar Empresas</h2>
              <button onClick={() => setShowQualifyModal(false)} className="text-gray-400 hover:text-gray-600"><X size={18} /></button>
            </div>
            <div className="bg-gray-50 rounded-xl px-4 py-3 text-sm text-gray-700 space-y-1">
              <div className="flex justify-between">
                <span>Empresas a qualificar</span>
                <span className="font-medium">{qualifyBatch.length}</span>
              </div>
              <div className="flex justify-between">
                <span>Custo total</span>
                <span className="font-medium text-purple-700">{qualifyBatch.length * 10} créditos</span>
              </div>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setShowQualifyModal(false)}
                className="flex-1 px-4 py-2 border border-gray-300 text-sm rounded-lg hover:bg-gray-50"
              >
                Cancelar
              </button>
              <button
                onClick={handleQualifyBatch}
                disabled={qualifyBatchLoading || qualifyBatch.length === 0}
                className="flex-1 px-4 py-2 bg-purple-600 text-white text-sm font-medium rounded-lg hover:bg-purple-700 disabled:opacity-50"
              >
                {qualifyBatchLoading ? "Qualificando..." : "Confirmar"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
