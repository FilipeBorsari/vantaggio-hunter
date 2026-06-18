"use client";

import { useState } from "react";
import { Brain, Search, MessageSquare, Sparkles, Copy, Check, Star } from "lucide-react";

interface CNAE {
  code: string;
  description: string;
}

function CNAEAssistant() {
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(false);
  const [cnaes, setCnaes] = useState<CNAE[]>([]);
  const [error, setError] = useState("");
  const [copiedCode, setCopiedCode] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setCnaes([]);
    setLoading(true);

    try {
      const res = await fetch("/api/intelligence/cnae", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ description }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error ?? "Erro ao buscar CNAEs");
        return;
      }
      setCnaes(data.cnaes ?? []);
    } catch {
      setError("Erro de conexão. Tente novamente.");
    } finally {
      setLoading(false);
    }
  }

  function copyCode(code: string) {
    navigator.clipboard.writeText(code);
    setCopiedCode(code);
    setTimeout(() => setCopiedCode(null), 2000);
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div className="bg-white border border-gray-200 rounded-xl p-5 flex flex-col gap-4">
        <div>
          <h2 className="text-base font-semibold text-gray-900 flex items-center gap-2">
            <Search size={16} className="text-indigo-600" />
            Encontrar CNAEs Ideais
          </h2>
          <p className="text-sm text-gray-500 mt-0.5">
            Descreva o negócio ou setor e receba sugestões de CNAEs mais adequados
          </p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">
              Descrição do Negócio
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Ex: Loja de roupas femininas online, delivery de comida, consultoria em marketing digital..."
              rows={5}
              required
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none"
            />
          </div>

          {error && (
            <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading || !description.trim()}
            className="inline-flex items-center justify-center gap-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-60 text-white font-medium px-5 py-2.5 rounded-lg text-sm transition-colors"
          >
            <Sparkles size={14} />
            {loading ? "Analisando..." : "Analisar CNAEs"}
          </button>
        </form>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl p-5 flex flex-col gap-4">
        <div>
          <h2 className="text-base font-semibold text-gray-900">CNAEs Sugeridos</h2>
          <p className="text-sm text-gray-500 mt-0.5">
            Lista de CNAEs recomendados com códigos e descrições
          </p>
        </div>

        {cnaes.length === 0 && !loading && (
          <div className="flex-1 flex flex-col items-center justify-center py-10 text-gray-400 gap-2">
            <Search size={32} className="opacity-30" />
            <p className="text-sm">Os CNAEs sugeridos aparecerão aqui após a análise</p>
          </div>
        )}

        {loading && (
          <div className="flex-1 flex flex-col items-center justify-center py-10 text-gray-400 gap-2">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
            <p className="text-sm">Consultando IA...</p>
          </div>
        )}

        {cnaes.length > 0 && (
          <ul className="flex flex-col gap-2">
            {cnaes.map((cnae) => (
              <li
                key={cnae.code}
                className="flex items-start gap-3 p-3 rounded-lg border border-gray-100 hover:border-indigo-200 hover:bg-indigo-50/30 transition-colors group"
              >
                <span className="font-mono text-sm font-semibold text-indigo-700 shrink-0 bg-indigo-50 px-2 py-0.5 rounded">
                  {cnae.code}
                </span>
                <span className="text-sm text-gray-700 flex-1 leading-snug">{cnae.description}</span>
                <button
                  onClick={() => copyCode(cnae.code)}
                  title="Copiar código"
                  className="shrink-0 opacity-0 group-hover:opacity-100 text-gray-400 hover:text-indigo-600 transition-all"
                >
                  {copiedCode === cnae.code ? (
                    <Check size={14} className="text-green-600" />
                  ) : (
                    <Copy size={14} />
                  )}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function TemplateGenerator() {
  const [type, setType] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ template: string; variables: string[]; tips: string[] } | null>(null);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setResult(null);
    setLoading(true);

    try {
      const res = await fetch("/api/intelligence/template", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ type }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error ?? "Erro ao gerar template");
        return;
      }
      setResult(data);
    } catch {
      setError("Erro de conexão. Tente novamente.");
    } finally {
      setLoading(false);
    }
  }

  function copyTemplate() {
    if (!result) return;
    navigator.clipboard.writeText(result.template);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div className="bg-white border border-gray-200 rounded-xl p-5 flex flex-col gap-4">
        <div>
          <h2 className="text-base font-semibold text-gray-900 flex items-center gap-2">
            <MessageSquare size={16} className="text-indigo-600" />
            Gerar Template de Mensagem
          </h2>
          <p className="text-sm text-gray-500 mt-0.5">
            Descreva o tipo de template que precisa para aprovação na Meta
          </p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">
              Tipo de Template
            </label>
            <textarea
              value={type}
              onChange={(e) => setType(e.target.value)}
              placeholder="Ex: Template de boas-vindas para e-commerce, mensagem de cobrança amigável, convite para evento, follow-up de vendas..."
              rows={5}
              required
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none"
            />
          </div>

          {error && (
            <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading || !type.trim()}
            className="inline-flex items-center justify-center gap-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-60 text-white font-medium px-5 py-2.5 rounded-lg text-sm transition-colors"
          >
            <Sparkles size={14} />
            {loading ? "Gerando..." : "Gerar Template"}
          </button>
        </form>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl p-5 flex flex-col gap-4">
        <div>
          <h2 className="text-base font-semibold text-gray-900">Template Gerado</h2>
          <p className="text-sm text-gray-500 mt-0.5">
            Template otimizado para aprovação na Meta
          </p>
        </div>

        {!result && !loading && (
          <div className="flex-1 flex flex-col items-center justify-center py-10 text-gray-400 gap-2">
            <MessageSquare size={32} className="opacity-30" />
            <p className="text-sm">O template gerado aparecerá aqui</p>
          </div>
        )}

        {loading && (
          <div className="flex-1 flex flex-col items-center justify-center py-10 text-gray-400 gap-2">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
            <p className="text-sm">Gerando template...</p>
          </div>
        )}

        {result && (
          <div className="flex flex-col gap-4">
            <div className="relative">
              <div className="font-mono text-sm bg-gray-50 border border-gray-200 rounded-lg p-4 whitespace-pre-wrap leading-relaxed text-gray-800">
                {result.template}
              </div>
              <button
                onClick={copyTemplate}
                className="absolute top-2 right-2 p-1.5 rounded-md bg-white border border-gray-200 text-gray-400 hover:text-indigo-600 hover:border-indigo-300 transition-colors"
                title="Copiar template"
              >
                {copied ? <Check size={14} className="text-green-600" /> : <Copy size={14} />}
              </button>
            </div>

            {result.variables && result.variables.length > 0 && (
              <div>
                <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">
                  Variáveis
                </p>
                <ul className="flex flex-col gap-1">
                  {result.variables.map((v, i) => (
                    <li key={i} className="text-sm text-gray-700 flex items-start gap-2">
                      <span className="font-mono text-indigo-600 shrink-0">{`{{${i + 1}}}`}</span>
                      {v}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {result.tips && result.tips.length > 0 && (
              <div>
                <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">
                  Dicas para aprovação
                </p>
                <ul className="flex flex-col gap-1">
                  {result.tips.map((tip, i) => (
                    <li key={i} className="text-sm text-gray-600 flex items-start gap-2">
                      <span className="text-green-500 shrink-0 mt-0.5">✓</span>
                      {tip}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

interface QualificationResult {
  qualification_id: string;
  cnpj: string;
  score: number;
  justification: string;
  model: string;
  credits_used: number;
  from_cache?: boolean;
}

function ScoreBadge({ score }: { score: number }) {
  const color =
    score > 70 ? "bg-green-50 text-green-700 border-green-200"
    : score >= 40 ? "bg-yellow-50 text-yellow-700 border-yellow-200"
    : "bg-red-50 text-red-700 border-red-200";
  return (
    <span className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-sm font-bold border ${color}`}>
      <Star size={13} />
      {score}/100
    </span>
  );
}

function QualifyTab() {
  const [cnpj, setCnpj] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<QualificationResult | null>(null);
  const [error, setError] = useState("");

  function sanitizeCNPJ(v: string) {
    return v.replace(/\D/g, "").slice(0, 14);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setResult(null);
    const raw = cnpj.replace(/\D/g, "");
    if (raw.length !== 14) {
      setError("CNPJ deve ter 14 dígitos.");
      return;
    }
    setLoading(true);
    try {
      const res = await fetch(`/api/ia/qualify/${raw}`, { method: "POST" });
      const data = await res.json();
      if (!res.ok) {
        if (res.status === 402) setError("Créditos insuficientes.");
        else if (res.status === 404) setError("Empresa não encontrada na base.");
        else setError(data.error ?? "Erro ao qualificar.");
        return;
      }
      setResult(data);
    } catch {
      setError("Erro de conexão. Tente novamente.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div className="bg-white border border-gray-200 rounded-xl p-5 flex flex-col gap-4">
        <div>
          <h2 className="text-base font-semibold text-gray-900 flex items-center gap-2">
            <Star size={16} className="text-purple-600" />
            Qualificar Empresa
          </h2>
          <p className="text-sm text-gray-500 mt-0.5">
            Receba um score de 0–100 com justificativa detalhada. Custa 10 créditos por CNPJ.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">CNPJ</label>
            <input
              type="text"
              value={cnpj}
              onChange={(e) => setCnpj(sanitizeCNPJ(e.target.value))}
              placeholder="00000000000000"
              maxLength={14}
              required
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-purple-500"
            />
          </div>

          {error && (
            <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading || cnpj.length !== 14}
            className="inline-flex items-center justify-center gap-2 bg-purple-600 hover:bg-purple-700 disabled:opacity-60 text-white font-medium px-5 py-2.5 rounded-lg text-sm transition-colors"
          >
            <Sparkles size={14} />
            {loading ? "Qualificando..." : "Qualificar (10 créditos)"}
          </button>
        </form>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl p-5 flex flex-col gap-4">
        <div>
          <h2 className="text-base font-semibold text-gray-900">Resultado da Qualificação</h2>
          <p className="text-sm text-gray-500 mt-0.5">
            Score e justificativa gerados por IA
          </p>
        </div>

        {!result && !loading && (
          <div className="flex-1 flex flex-col items-center justify-center py-10 text-gray-400 gap-2">
            <Star size={32} className="opacity-30" />
            <p className="text-sm">O resultado aparecerá aqui após a qualificação</p>
          </div>
        )}

        {loading && (
          <div className="flex-1 flex flex-col items-center justify-center py-10 text-gray-400 gap-2">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-purple-600" />
            <p className="text-sm">Consultando IA...</p>
          </div>
        )}

        {result && (
          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-3">
              <ScoreBadge score={result.score} />
              <div>
                <p className="text-xs text-gray-500 font-mono">{result.cnpj}</p>
                {result.from_cache && (
                  <p className="text-xs text-gray-400 mt-0.5">Resultado em cache · 0 créditos debitados</p>
                )}
                {!result.from_cache && (
                  <p className="text-xs text-gray-400 mt-0.5">{result.credits_used} crédito{result.credits_used !== 1 ? "s" : ""} debitado{result.credits_used !== 1 ? "s" : ""}</p>
                )}
              </div>
            </div>

            <div className="bg-gray-50 border border-gray-100 rounded-xl p-4">
              <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">Justificativa</p>
              <p className="text-sm text-gray-700 leading-relaxed">{result.justification}</p>
            </div>

            <p className="text-xs text-gray-400">Modelo: {result.model}</p>
          </div>
        )}
      </div>
    </div>
  );
}

export default function IntelligencePage() {
  const [tab, setTab] = useState<"cnae" | "template" | "qualify">("cnae");

  return (
    <div className="max-w-5xl mx-auto flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold text-gray-900 flex items-center gap-2">
          <Brain size={20} className="text-indigo-600" />
          Inteligência Artificial
        </h1>
        <p className="text-sm text-gray-500 mt-0.5">
          Assistentes inteligentes para CNAEs, templates de mensagem e qualificação de leads
        </p>
      </div>

      <div className="flex gap-1 p-1 bg-gray-100 rounded-xl w-fit">
        <button
          onClick={() => setTab("cnae")}
          className={`flex items-center gap-2 px-4 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            tab === "cnae" ? "bg-white text-gray-900 shadow-sm" : "text-gray-500 hover:text-gray-700"
          }`}
        >
          <Search size={14} />
          Assistente CNAE
        </button>
        <button
          onClick={() => setTab("template")}
          className={`flex items-center gap-2 px-4 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            tab === "template" ? "bg-white text-gray-900 shadow-sm" : "text-gray-500 hover:text-gray-700"
          }`}
        >
          <MessageSquare size={14} />
          Gerador de Templates
        </button>
        <button
          onClick={() => setTab("qualify")}
          className={`flex items-center gap-2 px-4 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            tab === "qualify" ? "bg-white text-gray-900 shadow-sm" : "text-gray-500 hover:text-gray-700"
          }`}
        >
          <Star size={14} />
          Qualificar CNPJ
        </button>
      </div>

      {tab === "cnae" && <CNAEAssistant />}
      {tab === "template" && <TemplateGenerator />}
      {tab === "qualify" && <QualifyTab />}
    </div>
  );
}
