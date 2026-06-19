"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Search } from "lucide-react";

const UF_OPTIONS = [
  "AC","AL","AM","AP","BA","CE","DF","ES","GO","MA","MG","MS","MT","PA",
  "PB","PE","PI","PR","RJ","RN","RO","RR","RS","SC","SE","SP","TO",
];

interface Company {
  cnpj: string;
  razao_social: string;
  nome_fantasia?: string;
  municipio?: string;
  uf: string;
  capital_social?: number;
  situacao_cadastral: number;
}

interface ListResponse {
  data: Company[];
  total: number;
  page: number;
  limit: number;
}

const SITUACAO_LABEL: Record<number, string> = {
  1: "Nula",
  2: "Ativa",
  3: "Suspensa",
  4: "Inapta",
  8: "Baixada",
};

function formatCNPJ(cnpj: string) {
  return cnpj.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, "$1.$2.$3/$4-$5");
}

function formatCurrency(v?: number) {
  if (v == null) return "—";
  return new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL", maximumFractionDigits: 0 }).format(v);
}

const inputClass = "w-full px-3 py-1.5 border border-v-border rounded-lg text-sm text-v-text bg-v-bg focus:outline-none focus:ring-2 focus:ring-v-accent";

export default function CompaniesPage() {
  const [filters, setFilters] = useState({ cnae: "", uf: "", status: "" });
  const [search, setSearch] = useState(filters);
  const [companies, setCompanies] = useState<Company[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const limit = 50;

  const fetchCompanies = useCallback(async (f: typeof filters, p: number) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: String(p), limit: String(limit) });
      if (f.cnae) params.set("cnae", f.cnae);
      if (f.uf) params.set("uf", f.uf);
      if (f.status) params.set("status", f.status);

      const res = await fetch(`/api/companies?${params}`);
      if (!res.ok) return;
      const json: ListResponse = await res.json();
      setCompanies(json.data ?? []);
      setTotal(json.total);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchCompanies(search, page);
  }, [search, page, fetchCompanies]);

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    setPage(1);
    setSearch({ ...filters });
  }

  const parentRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line react-hooks/incompatible-library -- useVirtualizer retorna funções não memorizáveis por design do TanStack Virtual
  const rowVirtualizer = useVirtualizer({
    count: companies.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 48,
    overscan: 10,
  });

  const totalPages = Math.ceil(total / limit);

  return (
    <div className="flex flex-col h-full gap-4">
      <div>
        <h1 className="text-xl font-semibold text-v-text">Lead Bank</h1>
        <p className="text-sm text-v-muted mt-0.5">
          {loading ? "Carregando..." : `${total.toLocaleString("pt-BR")} empresas encontradas`}
        </p>
      </div>

      {/* Filters */}
      <form onSubmit={handleSearch} className="flex flex-wrap gap-3 bg-v-card border border-v-card-border rounded-xl p-4">
        <div className="flex-1 min-w-40">
          <label className="block text-xs font-medium text-v-muted mb-1">CNAE</label>
          <input
            value={filters.cnae}
            onChange={(e) => setFilters((f) => ({ ...f, cnae: e.target.value }))}
            placeholder="Ex: 4520-0/01"
            className={inputClass}
          />
        </div>
        <div className="w-32">
          <label className="block text-xs font-medium text-v-muted mb-1">UF</label>
          <select
            value={filters.uf}
            onChange={(e) => setFilters((f) => ({ ...f, uf: e.target.value }))}
            className={inputClass}
          >
            <option value="">Todos</option>
            {UF_OPTIONS.map((uf) => (
              <option key={uf} value={uf}>{uf}</option>
            ))}
          </select>
        </div>
        <div className="w-36">
          <label className="block text-xs font-medium text-v-muted mb-1">Situação</label>
          <select
            value={filters.status}
            onChange={(e) => setFilters((f) => ({ ...f, status: e.target.value }))}
            className={inputClass}
          >
            <option value="">Todas</option>
            <option value="2">Ativa</option>
            <option value="3">Suspensa</option>
            <option value="4">Inapta</option>
            <option value="8">Baixada</option>
          </select>
        </div>
        <div className="flex items-end">
          <button
            type="submit"
            className="inline-flex items-center gap-2 bg-v-accent hover:bg-v-glow text-white font-medium px-4 py-1.5 rounded-lg text-sm transition-colors"
          >
            <Search size={14} />
            Buscar
          </button>
        </div>
      </form>

      {/* Table */}
      <div className="flex-1 bg-v-card border border-v-card-border rounded-xl overflow-hidden flex flex-col min-h-0">
        {/* Header */}
        <div className="grid grid-cols-[180px_1fr_160px_80px_120px_100px] gap-2 px-4 py-3 bg-v-bg border-b border-v-border text-xs font-medium text-v-muted shrink-0">
          <span>CNPJ</span>
          <span>Razão Social</span>
          <span>Município/UF</span>
          <span>UF</span>
          <span>Capital</span>
          <span>Situação</span>
        </div>

        {/* Virtualized rows */}
        <div ref={parentRef} className="flex-1 overflow-auto">
          {loading && companies.length === 0 ? (
            <div className="flex items-center justify-center h-40 text-sm text-v-muted">
              Carregando empresas...
            </div>
          ) : companies.length === 0 ? (
            <div className="flex items-center justify-center h-40 text-sm text-v-muted">
              Nenhuma empresa encontrada com os filtros aplicados
            </div>
          ) : (
            <div style={{ height: rowVirtualizer.getTotalSize(), position: "relative" }}>
              {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                const company = companies[virtualRow.index];
                return (
                  <div
                    key={virtualRow.key}
                    style={{
                      position: "absolute",
                      top: virtualRow.start,
                      width: "100%",
                      height: virtualRow.size,
                    }}
                    className="grid grid-cols-[180px_1fr_160px_80px_120px_100px] gap-2 items-center px-4 border-b border-v-card-border hover:bg-v-border/30 text-sm"
                  >
                    <span className="font-mono text-xs text-v-muted truncate">
                      {formatCNPJ(company.cnpj)}
                    </span>
                    <span className="font-medium text-v-text truncate" title={company.razao_social}>
                      {company.razao_social}
                    </span>
                    <span className="text-v-text/60 truncate">{company.municipio ?? "—"}</span>
                    <span className="text-v-text/60">{company.uf}</span>
                    <span className="text-v-text/60 tabular-nums">
                      {formatCurrency(company.capital_social)}
                    </span>
                    <span>
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                          company.situacao_cadastral === 2
                            ? "bg-green-900/30 text-green-400"
                            : "bg-v-border text-v-muted"
                        }`}
                      >
                        {SITUACAO_LABEL[company.situacao_cadastral] ?? company.situacao_cadastral}
                      </span>
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-v-border shrink-0">
            <span className="text-xs text-v-muted">
              Página {page} de {totalPages}
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
