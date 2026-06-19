"use client";

import { useEffect, useState, useCallback } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import {
  Activity,
  TrendingUp,
  Users,
  Zap,
} from "lucide-react";
import Link from "next/link";

// ─── Types ───────────────────────────────────────────────────────────────────

interface KPIs {
  period: string;
  credits_consumed: number;
  credits_purchased: number;
  leads_extracted: number;
  leads_qualified: number;
  leads_exported: number;
  conversion_rate: number;
  searches_count: number;
}

interface DailyPoint {
  date: string;
  credits: number;
  leads: number;
}

interface TopCNAE {
  cnae_code: string;
  description: string;
  leads: number;
}

interface FunnelStage {
  name: string;
  count: number;
}

interface FunnelResponse {
  stages: FunnelStage[];
}

interface RecentSearch {
  id: string;
  mode: string;
  status: string;
  result_count?: number;
  created_at: string;
  filters: {
    cnaes?: string[];
    uf?: string;
    city?: string;
  };
  query_text?: string;
}

// ─── Period selector ─────────────────────────────────────────────────────────

const PERIODS = [
  { label: "7 dias", value: "7d" },
  { label: "30 dias", value: "30d" },
  { label: "90 dias", value: "90d" },
];

// ─── Helpers ─────────────────────────────────────────────────────────────────

function fmt(n: number) {
  return n.toLocaleString("pt-BR");
}

function fmtPct(n: number) {
  return (n * 100).toFixed(2).replace(".", ",") + "%";
}

function fmtDate(iso: string) {
  return new Intl.DateTimeFormat("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(iso));
}

function fmtChartDate(iso: string) {
  const [, m, d] = iso.split("-");
  return `${d}/${m}`;
}

const STATUS_LABEL: Record<string, string> = {
  done: "Concluída",
  processing: "Processando",
  queued: "Na fila",
  failed: "Falha",
};

const STATUS_COLOR: Record<string, string> = {
  done: "text-green-400 bg-green-900/30",
  processing: "text-blue-400 bg-blue-900/30",
  queued: "text-yellow-400 bg-yellow-900/30",
  failed: "text-red-400 bg-red-900/30",
};

// ─── KPI Card ────────────────────────────────────────────────────────────────

interface KPICardProps {
  label: string;
  value: string;
  icon: React.ElementType;
  color: string;
}

function KPICard({ label, value, icon: Icon, color }: KPICardProps) {
  return (
    <div className="bg-v-card border border-v-card-border rounded-xl p-5 flex items-center gap-4">
      <div className={`p-3 rounded-xl ${color}`}>
        <Icon size={22} />
      </div>
      <div>
        <p className="text-xs text-v-muted uppercase tracking-wide font-medium">{label}</p>
        <p className="text-2xl font-bold tabular-nums text-v-text">{value}</p>
      </div>
    </div>
  );
}

// ─── Funnel bar ──────────────────────────────────────────────────────────────

function FunnelBars({ stages }: { stages: FunnelStage[] }) {
  const max = stages[0]?.count ?? 1;
  return (
    <div className="flex flex-col gap-3">
      {stages.map((stage, i) => {
        const pct = max > 0 ? (stage.count / max) * 100 : 0;
        const conversion =
          i > 0 && stages[i - 1].count > 0
            ? ((stage.count / stages[i - 1].count) * 100).toFixed(1)
            : null;
        return (
          <div key={stage.name} className="flex flex-col gap-1">
            <div className="flex items-center justify-between text-sm">
              <span className="font-medium text-v-text/80">{stage.name}</span>
              <span className="text-v-muted tabular-nums">
                {fmt(stage.count)}
                {conversion != null && (
                  <span className="ml-2 text-xs text-v-accent">({conversion}%)</span>
                )}
              </span>
            </div>
            <div className="h-3 bg-v-border rounded-full overflow-hidden">
              <div
                className="h-full bg-v-accent rounded-full transition-all duration-500"
                style={{ width: `${pct}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ─── Page ────────────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const [period, setPeriod] = useState("30d");
  const [kpis, setKpis] = useState<KPIs | null>(null);
  const [daily, setDaily] = useState<DailyPoint[]>([]);
  const [topCnaes, setTopCnaes] = useState<TopCNAE[]>([]);
  const [funnel, setFunnel] = useState<FunnelResponse | null>(null);
  const [recentSearches, setRecentSearches] = useState<RecentSearch[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchAll = useCallback(async (p: string) => {
    setLoading(true);
    const qs = `period=${p}`;
    try {
      const [kpisRes, dailyRes, cnaesRes, funnelRes, searchesRes] =
        await Promise.all([
          fetch(`/api/analytics/kpis?${qs}`).then((r) => (r.ok ? r.json() : null)),
          fetch(`/api/analytics/daily-consumption?${qs}`).then((r) =>
            r.ok ? r.json() : []
          ),
          fetch(`/api/analytics/top-cnaes?${qs}&limit=10`).then((r) =>
            r.ok ? r.json() : []
          ),
          fetch(`/api/analytics/funnel?${qs}`).then((r) => (r.ok ? r.json() : null)),
          fetch(`/api/analytics/searches?page=1&limit=5`).then((r) =>
            r.ok ? r.json() : null
          ),
        ]);
      setKpis(kpisRes);
      setDaily(dailyRes ?? []);
      setTopCnaes(cnaesRes ?? []);
      setFunnel(funnelRes);
      setRecentSearches(searchesRes?.data?.slice(0, 5) ?? []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAll(period);
  }, [period, fetchAll]);

  return (
    <div className="flex flex-col gap-6 max-w-6xl">
      {/* Header + period selector */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-semibold text-v-text">Dashboard</h1>
          <p className="text-sm text-v-muted mt-0.5">
            KPIs e funil de conversão da sua organização.
          </p>
        </div>
        <div className="flex gap-1 bg-v-border rounded-xl p-1">
          {PERIODS.map((p) => (
            <button
              key={p.value}
              onClick={() => setPeriod(p.value)}
              className={`px-4 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                period === p.value
                  ? "bg-v-accent text-white"
                  : "text-v-muted hover:text-v-text"
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <KPICard
          label="Créditos consumidos"
          value={loading ? "—" : fmt(kpis?.credits_consumed ?? 0)}
          icon={Zap}
          color="bg-amber-900/30 text-amber-400"
        />
        <KPICard
          label="Leads extraídos"
          value={loading ? "—" : fmt(kpis?.leads_extracted ?? 0)}
          icon={Users}
          color="bg-v-accent/10 text-v-accent"
        />
        <KPICard
          label="Leads qualificados"
          value={loading ? "—" : fmt(kpis?.leads_qualified ?? 0)}
          icon={Activity}
          color="bg-green-900/30 text-green-400"
        />
        <KPICard
          label="Taxa de conversão"
          value={loading ? "—" : fmtPct(kpis?.conversion_rate ?? 0)}
          icon={TrendingUp}
          color="bg-rose-900/30 text-rose-400"
        />
      </div>

      {/* Daily chart + Funnel */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Daily consumption chart */}
        <div className="lg:col-span-2 bg-v-card border border-v-card-border rounded-xl p-5">
          <h2 className="text-sm font-semibold text-v-text/80 mb-4">
            Consumo diário
          </h2>
          {loading || daily.length === 0 ? (
            <div className="h-52 flex items-center justify-center text-sm text-v-muted">
              {loading ? "Carregando..." : "Sem dados no período"}
            </div>
          ) : (
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={daily} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1F1F1F" />
                <XAxis
                  dataKey="date"
                  tickFormatter={fmtChartDate}
                  tick={{ fontSize: 11, fill: "#737373" }}
                  interval="preserveStartEnd"
                />
                <YAxis tick={{ fontSize: 11, fill: "#737373" }} />
                <Tooltip
                  contentStyle={{ backgroundColor: "#0c0b0a", border: "1px solid #1F1F1F", borderRadius: 8 }}
                  labelStyle={{ color: "#737373" }}
                  itemStyle={{ color: "#FFFFFF" }}
                  formatter={(v, name) => [
                    fmt(Number(v ?? 0)),
                    name === "credits" ? "Créditos" : "Leads",
                  ]}
                  labelFormatter={(l) => fmtChartDate(String(l ?? ""))}
                />
                <Legend
                  formatter={(v) => (v === "credits" ? "Créditos" : "Leads")}
                  wrapperStyle={{ fontSize: 12, color: "#737373" }}
                />
                <Line
                  type="monotone"
                  dataKey="credits"
                  stroke="#E8621A"
                  strokeWidth={2}
                  dot={false}
                />
                <Line
                  type="monotone"
                  dataKey="leads"
                  stroke="#10b981"
                  strokeWidth={2}
                  dot={false}
                />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* Funnel */}
        <div className="bg-v-card border border-v-card-border rounded-xl p-5">
          <h2 className="text-sm font-semibold text-v-text/80 mb-4">
            Funil de conversão
          </h2>
          {loading || !funnel ? (
            <div className="h-40 flex items-center justify-center text-sm text-v-muted">
              {loading ? "Carregando..." : "Sem dados"}
            </div>
          ) : (
            <FunnelBars stages={funnel.stages} />
          )}
        </div>
      </div>

      {/* Top CNAEs + Recent searches */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Top CNAEs */}
        <div className="bg-v-card border border-v-card-border rounded-xl overflow-hidden">
          <div className="px-5 py-4 border-b border-v-border">
            <h2 className="text-sm font-semibold text-v-text/80">Top CNAEs buscados</h2>
          </div>
          {loading ? (
            <div className="flex items-center justify-center h-32 text-sm text-v-muted">
              Carregando...
            </div>
          ) : topCnaes.length === 0 ? (
            <div className="flex items-center justify-center h-32 text-sm text-v-muted">
              Sem dados no período
            </div>
          ) : (
            <div className="divide-y divide-v-card-border">
              <div className="grid grid-cols-[80px_1fr_64px] gap-2 px-5 py-2 text-xs font-medium text-v-muted bg-v-bg/50">
                <span>CNAE</span>
                <span>Descrição</span>
                <span className="text-right">Leads</span>
              </div>
              {topCnaes.map((c) => (
                <div
                  key={c.cnae_code}
                  className="grid grid-cols-[80px_1fr_64px] gap-2 px-5 py-2.5 text-sm items-center"
                >
                  <span className="font-mono text-xs text-v-muted">{c.cnae_code}</span>
                  <span
                    className="text-v-text/70 truncate text-xs"
                    title={c.description}
                  >
                    {c.description}
                  </span>
                  <span className="text-right font-semibold tabular-nums text-v-accent">
                    {fmt(c.leads)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Recent searches */}
        <div className="bg-v-card border border-v-card-border rounded-xl overflow-hidden">
          <div className="px-5 py-4 border-b border-v-border flex items-center justify-between">
            <h2 className="text-sm font-semibold text-v-text/80">Buscas recentes</h2>
            <Link
              href="/search/history"
              className="text-xs text-v-accent hover:underline"
            >
              Ver todas
            </Link>
          </div>
          {loading ? (
            <div className="flex items-center justify-center h-32 text-sm text-v-muted">
              Carregando...
            </div>
          ) : recentSearches.length === 0 ? (
            <div className="flex items-center justify-center h-32 text-sm text-v-muted">
              Nenhuma busca ainda
            </div>
          ) : (
            <div className="divide-y divide-v-card-border">
              {recentSearches.map((s) => {
                const label =
                  s.query_text
                    ? s.query_text.slice(0, 40) + (s.query_text.length > 40 ? "…" : "")
                    : s.filters.cnaes?.join(", ") || "—";
                return (
                  <Link
                    key={s.id}
                    href={`/search/${s.id}`}
                    className="flex items-center gap-3 px-5 py-3 hover:bg-v-border/30 transition-colors"
                  >
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-v-text/80 truncate">{label}</p>
                      <p className="text-xs text-v-muted mt-0.5">{fmtDate(s.created_at)}</p>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      {s.result_count != null && (
                        <span className="text-xs text-v-muted tabular-nums">
                          {fmt(s.result_count)} leads
                        </span>
                      )}
                      <span
                        className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                          STATUS_COLOR[s.status] ?? "text-v-muted bg-v-border"
                        }`}
                      >
                        {STATUS_LABEL[s.status] ?? s.status}
                      </span>
                    </div>
                  </Link>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
