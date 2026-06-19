"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { useRouter } from "next/navigation";
import { Search, Sparkles } from "lucide-react";

const UF_OPTIONS = [
  "AC","AL","AM","AP","BA","CE","DF","ES","GO","MA","MG","MS","MT","PA",
  "PB","PE","PI","PR","RJ","RN","RO","RR","RS","SC","SE","SP","TO",
];

interface CNAE {
  code: string;
  description: string;
}

interface StructuredFilters {
  cnaes: string[];
  uf: string;
  city: string;
  capital_min: string;
  status: string;
  max_results: string;
}

function CNAEMultiSelect({
  selected,
  onChange,
}: {
  selected: string[];
  onChange: (v: string[]) => void;
}) {
  const [query, setQuery] = useState("");
  const [suggestions, setSuggestions] = useState<CNAE[]>([]);
  const [open, setOpen] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchSuggestions = useCallback(async (q: string) => {
    if (!q) { setSuggestions([]); return; }
    const res = await fetch(`/api/cnaes?q=${encodeURIComponent(q)}`);
    if (res.ok) setSuggestions(await res.json());
  }, []);

  function handleInput(e: React.ChangeEvent<HTMLInputElement>) {
    const v = e.target.value;
    setQuery(v);
    setOpen(true);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => fetchSuggestions(v), 300);
  }

  function addCNAE(code: string) {
    if (!selected.includes(code)) onChange([...selected, code]);
    setQuery("");
    setSuggestions([]);
    setOpen(false);
  }

  function removeCNAE(code: string) {
    onChange(selected.filter((c) => c !== code));
  }

  return (
    <div className="relative">
      <label className="block text-xs font-medium text-v-muted mb-1">CNAEs</label>
      <div className="flex flex-wrap gap-1 min-h-[36px] px-2 py-1 border border-v-border rounded-lg bg-v-bg focus-within:ring-2 focus-within:ring-v-accent">
        {selected.map((code) => (
          <span
            key={code}
            className="inline-flex items-center gap-1 px-2 py-0.5 bg-v-accent/10 text-v-accent rounded text-xs font-mono"
          >
            {code}
            <button type="button" onClick={() => removeCNAE(code)} className="hover:text-v-accent-2 font-bold">×</button>
          </span>
        ))}
        <input
          value={query}
          onChange={handleInput}
          onFocus={() => setOpen(true)}
          onBlur={() => setTimeout(() => setOpen(false), 200)}
          placeholder="Buscar CNAE..."
          className="flex-1 min-w-24 text-sm outline-none bg-transparent text-v-text placeholder:text-v-muted"
        />
      </div>
      {open && suggestions.length > 0 && (
        <ul className="absolute z-10 mt-1 w-full bg-v-card border border-v-border rounded-lg shadow-lg max-h-48 overflow-auto">
          {suggestions.map((c) => (
            <li key={c.code}>
              <button
                type="button"
                onMouseDown={() => addCNAE(c.code)}
                className="w-full text-left px-3 py-2 text-sm hover:bg-v-border/40 flex gap-2"
              >
                <span className="font-mono text-v-accent shrink-0">{c.code}</span>
                <span className="text-v-text/70 truncate">{c.description}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

const inputClass = "w-full px-3 py-1.5 border border-v-border rounded-lg text-sm text-v-text bg-v-bg focus:outline-none focus:ring-2 focus:ring-v-accent";

export default function SearchPage() {
  const router = useRouter();
  const [mode, setMode] = useState<"structured" | "semantic">("structured");
  const [query, setQuery] = useState("");
  const [filters, setFilters] = useState<StructuredFilters>({
    cnaes: [], uf: "", city: "", capital_min: "", status: "", max_results: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [estimate, setEstimate] = useState<number | null>(null);
  const [estimating, setEstimating] = useState(false);
  const estimateTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchEstimate = useCallback(async (
    currentMode: "structured" | "semantic",
    currentQuery: string,
    currentFilters: StructuredFilters,
  ) => {
    const hasData = currentMode === "semantic"
      ? currentQuery.trim().length > 3
      : currentFilters.cnaes.length > 0 || !!currentFilters.uf || !!currentFilters.status;
    if (!hasData) { setEstimate(null); return; }

    setEstimating(true);
    try {
      const body = currentMode === "semantic"
        ? {
            mode: "semantic",
            query: currentQuery,
            filters: {
              uf: currentFilters.uf || undefined,
              status: currentFilters.status ? parseInt(currentFilters.status) : undefined,
              max_results: currentFilters.max_results ? parseInt(currentFilters.max_results) : undefined,
            },
          }
        : {
            mode: "structured",
            filters: {
              cnaes: currentFilters.cnaes.length > 0 ? currentFilters.cnaes : undefined,
              uf: currentFilters.uf || undefined,
              city: currentFilters.city || undefined,
              capital_min: currentFilters.capital_min ? parseFloat(currentFilters.capital_min) : undefined,
              status: currentFilters.status ? parseInt(currentFilters.status) : undefined,
              max_results: currentFilters.max_results ? parseInt(currentFilters.max_results) : undefined,
            },
          };
      const res = await fetch("/api/searches/estimate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (res.ok) {
        const data = await res.json();
        setEstimate(data.estimate ?? null);
      }
    } catch {
      // estimativa é opcional, ignora falha silenciosamente
    } finally {
      setEstimating(false);
    }
  }, []);

  useEffect(() => {
    if (estimateTimer.current) clearTimeout(estimateTimer.current);
    estimateTimer.current = setTimeout(() => fetchEstimate(mode, query, filters), 500);
    return () => { if (estimateTimer.current) clearTimeout(estimateTimer.current); };
  }, [mode, query, filters, fetchEstimate]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const body =
        mode === "semantic"
          ? {
              mode: "semantic",
              query,
              filters: {
                uf: filters.uf || undefined,
                status: filters.status ? parseInt(filters.status) : undefined,
                max_results: filters.max_results ? parseInt(filters.max_results) : undefined,
              },
            }
          : {
              mode: "structured",
              filters: {
                cnaes: filters.cnaes.length > 0 ? filters.cnaes : undefined,
                uf: filters.uf || undefined,
                city: filters.city || undefined,
                capital_min: filters.capital_min ? parseFloat(filters.capital_min) : undefined,
                status: filters.status ? parseInt(filters.status) : undefined,
                max_results: filters.max_results ? parseInt(filters.max_results) : undefined,
              },
            };

      const res = await fetch("/api/searches", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error ?? "Erro ao criar busca");
        return;
      }

      const data = await res.json();
      router.push(`/search/${data.search_id}`);
    } catch {
      setError("Erro de conexão. Tente novamente.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="max-w-2xl mx-auto flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold text-v-text">Nova Busca</h1>
        <p className="text-sm text-v-muted mt-0.5">
          Encontre empresas por filtros estruturados ou por descrição em linguagem natural.
        </p>
      </div>

      {/* Mode toggle */}
      <div className="flex gap-1 p-1 bg-v-border rounded-xl w-fit">
        <button
          type="button"
          onClick={() => setMode("structured")}
          className={`flex items-center gap-2 px-4 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            mode === "structured" ? "bg-v-accent text-white" : "text-v-muted hover:text-v-text"
          }`}
        >
          <Search size={14} />
          Filtros
        </button>
        <button
          type="button"
          onClick={() => setMode("semantic")}
          className={`flex items-center gap-2 px-4 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            mode === "semantic" ? "bg-v-accent text-white" : "text-v-muted hover:text-v-text"
          }`}
        >
          <Sparkles size={14} />
          Busca com IA
        </button>
      </div>

      <form onSubmit={handleSubmit} className="bg-v-card border border-v-card-border rounded-xl p-5 flex flex-col gap-4">
        {mode === "semantic" ? (
          <>
            <div>
              <label className="block text-xs font-medium text-v-muted mb-1">
                Descreva o perfil de empresa que você procura
              </label>
              <textarea
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Ex: mecânicas de alto faturamento em São Paulo com presença digital"
                rows={3}
                required
                className="w-full px-3 py-2 border border-v-border rounded-lg text-sm text-v-text bg-v-bg placeholder:text-v-muted focus:outline-none focus:ring-2 focus:ring-v-accent resize-none"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-v-muted mb-1">UF (opcional)</label>
                <select
                  value={filters.uf}
                  onChange={(e) => setFilters((f) => ({ ...f, uf: e.target.value }))}
                  className={inputClass}
                >
                  <option value="">Todos os estados</option>
                  {UF_OPTIONS.map((uf) => <option key={uf} value={uf}>{uf}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-v-muted mb-1">Situação (opcional)</label>
                <select
                  value={filters.status}
                  onChange={(e) => setFilters((f) => ({ ...f, status: e.target.value }))}
                  className={inputClass}
                >
                  <option value="">Todas</option>
                  <option value="2">Ativa</option>
                  <option value="4">Inapta</option>
                  <option value="8">Baixada</option>
                </select>
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium text-v-muted mb-1">Limite de leads (opcional)</label>
              <input
                type="number"
                min="1"
                max="10000"
                value={filters.max_results}
                onChange={(e) => setFilters((f) => ({ ...f, max_results: e.target.value }))}
                placeholder="Ex: 200"
                className={inputClass}
              />
              <p className="text-xs text-v-muted mt-1">Deixe em branco para retornar todos os leads encontrados (máx. 10 000).</p>
            </div>
          </>
        ) : (
          <>
            <CNAEMultiSelect
              selected={filters.cnaes}
              onChange={(v) => setFilters((f) => ({ ...f, cnaes: v }))}
            />
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-v-muted mb-1">UF</label>
                <select
                  value={filters.uf}
                  onChange={(e) => setFilters((f) => ({ ...f, uf: e.target.value }))}
                  className={inputClass}
                >
                  <option value="">Todos os estados</option>
                  {UF_OPTIONS.map((uf) => <option key={uf} value={uf}>{uf}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-v-muted mb-1">Cidade</label>
                <input
                  value={filters.city}
                  onChange={(e) => setFilters((f) => ({ ...f, city: e.target.value }))}
                  placeholder="Ex: São Paulo"
                  className={inputClass}
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-v-muted mb-1">Capital Mínimo (R$)</label>
                <input
                  type="number"
                  min="0"
                  value={filters.capital_min}
                  onChange={(e) => setFilters((f) => ({ ...f, capital_min: e.target.value }))}
                  placeholder="Ex: 100000"
                  className={inputClass}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-v-muted mb-1">Situação</label>
                <select
                  value={filters.status}
                  onChange={(e) => setFilters((f) => ({ ...f, status: e.target.value }))}
                  className={inputClass}
                >
                  <option value="">Todas</option>
                  <option value="2">Ativa</option>
                  <option value="4">Inapta</option>
                  <option value="8">Baixada</option>
                </select>
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium text-v-muted mb-1">Limite de leads (opcional)</label>
              <input
                type="number"
                min="1"
                max="10000"
                value={filters.max_results}
                onChange={(e) => setFilters((f) => ({ ...f, max_results: e.target.value }))}
                placeholder="Ex: 200"
                className={inputClass}
              />
              <p className="text-xs text-v-muted mt-1">Deixe em branco para retornar todos os leads encontrados (máx. 10 000).</p>
            </div>
          </>
        )}

        {error && (
          <p className="text-sm text-red-400 bg-red-900/30 border border-red-900/50 rounded-lg px-3 py-2">{error}</p>
        )}

        <div className="rounded-lg bg-v-bg border border-v-border px-4 py-3 flex items-center justify-between">
          <div>
            <p className="text-xs text-v-muted uppercase tracking-wide font-medium mb-0.5">Gasto estimado em créditos</p>
            <p className="text-sm font-semibold text-v-text">
              {estimating
                ? <span className="text-v-muted font-normal">calculando...</span>
                : estimate !== null
                  ? `~${estimate.toLocaleString("pt-BR")} créditos`
                  : <span className="text-v-muted font-normal">—</span>
              }
            </p>
          </div>
          <div className="flex items-center gap-4">
            <a href="/search/history" className="text-sm text-v-muted hover:text-v-text underline-offset-2 hover:underline">
              Ver histórico
            </a>
            <button
              type="submit"
              disabled={loading}
              className="inline-flex items-center gap-2 bg-v-accent hover:bg-v-glow disabled:opacity-60 text-white font-medium px-5 py-2 rounded-lg text-sm transition-colors"
            >
              {mode === "semantic" ? <Sparkles size={14} /> : <Search size={14} />}
              {loading ? "Processando..." : mode === "semantic" ? "Buscar com IA" : "Buscar"}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
