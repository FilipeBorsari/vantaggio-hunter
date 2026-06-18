"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronRight, RefreshCw } from "lucide-react";

interface ExportErrorEntry {
  cnpj: string;
  error: string;
  attempt: number;
}

interface ExportJob {
  id: string;
  status: "pending" | "processing" | "done" | "partial" | "failed";
  crm_type: string;
  total_count: number;
  success_count: number;
  fail_count: number;
  error_log: ExportErrorEntry[];
  created_at: string;
  done_at?: string;
}

interface ExportListResponse {
  data: ExportJob[];
  total: number;
}

const STATUS_CONFIG: Record<ExportJob["status"], { label: string; className: string }> = {
  pending:    { label: "Aguardando",   className: "bg-yellow-50 text-yellow-700" },
  processing: { label: "Processando",  className: "bg-blue-50 text-blue-700" },
  done:       { label: "Concluído",    className: "bg-green-50 text-green-700" },
  partial:    { label: "Parcial",      className: "bg-orange-50 text-orange-700" },
  failed:     { label: "Falhou",       className: "bg-red-50 text-red-700" },
};

function formatDate(iso: string) {
  return new Date(iso).toLocaleString("pt-BR", {
    day: "2-digit", month: "2-digit", year: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

function formatCNPJ(cnpj: string) {
  return cnpj.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, "$1.$2.$3/$4-$5");
}

function StatusBadge({ status }: { status: ExportJob["status"] }) {
  const cfg = STATUS_CONFIG[status];
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${cfg.className}`}>
      {status === "processing" && <RefreshCw size={10} className="animate-spin" />}
      {cfg.label}
    </span>
  );
}

function ExportRow({ job }: { job: ExportJob }) {
  const [expanded, setExpanded] = useState(false);
  const hasErrors = job.error_log && job.error_log.length > 0;

  return (
    <>
      <tr
        className={`border-b border-gray-100 text-sm hover:bg-gray-50 ${hasErrors ? "cursor-pointer" : ""}`}
        onClick={() => hasErrors && setExpanded((v) => !v)}
      >
        <td className="px-4 py-3 text-gray-500 tabular-nums text-xs">
          {formatDate(job.created_at)}
        </td>
        <td className="px-4 py-3 font-mono text-xs text-gray-500">{job.crm_type}</td>
        <td className="px-4 py-3 tabular-nums text-gray-700">{job.total_count}</td>
        <td className="px-4 py-3 tabular-nums text-green-700">{job.success_count}</td>
        <td className="px-4 py-3 tabular-nums text-red-600">{job.fail_count}</td>
        <td className="px-4 py-3"><StatusBadge status={job.status} /></td>
        <td className="px-4 py-3 text-gray-400">
          {hasErrors ? (
            expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />
          ) : null}
        </td>
      </tr>
      {expanded && hasErrors && (
        <tr className="bg-red-50">
          <td colSpan={7} className="px-6 py-3">
            <p className="text-xs font-medium text-red-700 mb-2">Erros detalhados</p>
            <div className="space-y-1">
              {job.error_log.map((e, i) => (
                <div key={i} className="flex gap-3 text-xs text-red-600">
                  <span className="font-mono">{formatCNPJ(e.cnpj)}</span>
                  <span className="text-red-400">tentativa {e.attempt}</span>
                  <span>{e.error}</span>
                </div>
              ))}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

export default function ExportsPage() {
  const [data, setData] = useState<ExportListResponse | null>(null);
  const [page, setPage] = useState(1);
  const limit = 20;
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchExports = useCallback(async (p: number) => {
    const res = await fetch(`/api/exports?page=${p}&limit=${limit}`);
    if (!res.ok) return null;
    const json: ExportListResponse = await res.json();
    setData(json);
    return json;
  }, []);

  useEffect(() => {
    fetchExports(page);

    // Poll while any job is still in-flight.
    pollRef.current = setInterval(async () => {
      const json = await fetchExports(page);
      const hasActive = json?.data.some(
        (j) => j.status === "pending" || j.status === "processing"
      );
      if (!hasActive && pollRef.current) {
        clearInterval(pollRef.current);
      }
    }, 3000);

    return () => { if (pollRef.current) clearInterval(pollRef.current); };
  }, [fetchExports, page]);

  const jobs = data?.data ?? [];
  const totalPages = data?.total != null ? Math.ceil(data.total / limit) : 0;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-gray-900">Exportações CRM</h1>
          <p className="text-sm text-gray-500 mt-0.5">Histórico de envios para o Chatwoot</p>
        </div>
        <a
          href="/settings/crm"
          className="text-sm text-indigo-600 hover:text-indigo-800 font-medium"
        >
          Configurar integração →
        </a>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
        {jobs.length === 0 && data !== null ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            Nenhuma exportação ainda
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-gray-50 border-b border-gray-100 text-xs font-medium text-gray-600">
                <th className="px-4 py-3 text-left">Data</th>
                <th className="px-4 py-3 text-left">CRM</th>
                <th className="px-4 py-3 text-left">Total</th>
                <th className="px-4 py-3 text-left">Sucesso</th>
                <th className="px-4 py-3 text-left">Falhas</th>
                <th className="px-4 py-3 text-left">Status</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {jobs.map((job) => <ExportRow key={job.id} job={job} />)}
            </tbody>
          </table>
        )}
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-500">
            Página {page} de {totalPages}
          </span>
          <div className="flex gap-2">
            <button
              disabled={page === 1}
              onClick={() => setPage((p) => p - 1)}
              className="px-3 py-1 border border-gray-300 rounded-lg disabled:opacity-40 hover:bg-gray-50"
            >
              Anterior
            </button>
            <button
              disabled={page === totalPages}
              onClick={() => setPage((p) => p + 1)}
              className="px-3 py-1 border border-gray-300 rounded-lg disabled:opacity-40 hover:bg-gray-50"
            >
              Próxima
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
