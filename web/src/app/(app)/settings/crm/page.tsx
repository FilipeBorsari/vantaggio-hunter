"use client";

import { useEffect, useState } from "react";
import { CheckCircle, XCircle } from "lucide-react";

interface CRMIntegration {
  id: string;
  crm_type: string;
  base_url: string;
  inbox_id?: number;
  account_id: number;
  is_active: boolean;
  created_at: string;
}

interface FormState {
  base_url: string;
  api_key: string;
  inbox_id: string;
  account_id: string;
}

const inputClass = "w-full px-3 py-2 border border-v-border rounded-lg text-sm text-v-text bg-v-bg placeholder:text-v-muted focus:outline-none focus:ring-2 focus:ring-v-accent";

function Field({
  label,
  id,
  type = "text",
  value,
  onChange,
  placeholder,
  required,
  hint,
}: {
  label: string;
  id: string;
  type?: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  required?: boolean;
  hint?: string;
}) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-v-text/80 mb-1">
        {label} {required && <span className="text-red-400">*</span>}
      </label>
      <input
        id={id}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        required={required}
        className={inputClass}
      />
      {hint && <p className="mt-1 text-xs text-v-muted">{hint}</p>}
    </div>
  );
}

export default function CRMSettingsPage() {
  const [integration, setIntegration] = useState<CRMIntegration | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [form, setForm] = useState<FormState>({
    base_url: "",
    api_key: "",
    inbox_id: "",
    account_id: "1",
  });
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/crm/integrations")
      .then(async (res) => {
        if (res.status === 404) {
          setNotFound(true);
          return;
        }
        if (res.ok) {
          const data: CRMIntegration = await res.json();
          setIntegration(data);
          setForm({
            base_url: data.base_url,
            api_key: "",
            inbox_id: data.inbox_id != null ? String(data.inbox_id) : "",
            account_id: String(data.account_id ?? 1),
          });
        }
      })
      .catch(() => setNotFound(true));
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSaved(false);
    setSaving(true);

    try {
      const body: Record<string, unknown> = {
        crm_type: "chatwoot",
        base_url: form.base_url.replace(/\/$/, ""),
        api_key: form.api_key,
        account_id: Number(form.account_id) || 1,
      };
      if (form.inbox_id) body.inbox_id = Number(form.inbox_id);

      const res = await fetch("/api/crm/integrations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error ?? "Erro ao salvar integração");
        return;
      }

      const data: CRMIntegration = await res.json();
      setIntegration(data);
      setNotFound(false);
      setSaved(true);
      setForm((f) => ({ ...f, api_key: "" }));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="max-w-xl flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold text-v-text">Integração CRM</h1>
        <p className="text-sm text-v-muted mt-0.5">
          Configure o Chatwoot para receber os leads exportados do Vantaggio Hunter.
        </p>
      </div>

      {integration && (
        <div className="flex items-center gap-2 px-4 py-3 bg-green-900/30 border border-green-800 rounded-xl text-sm text-green-400">
          <CheckCircle size={16} />
          Integração ativa — {integration.base_url}
        </div>
      )}
      {notFound && !integration && (
        <div className="flex items-center gap-2 px-4 py-3 bg-yellow-900/30 border border-yellow-800 rounded-xl text-sm text-yellow-400">
          <XCircle size={16} />
          Nenhuma integração configurada ainda
        </div>
      )}

      <form onSubmit={handleSubmit} className="bg-v-card border border-v-card-border rounded-xl p-6 flex flex-col gap-4">
        <Field
          label="URL base do Chatwoot"
          id="base_url"
          value={form.base_url}
          onChange={(v) => setForm((f) => ({ ...f, base_url: v }))}
          placeholder="https://app.chatwoot.com"
          required
          hint="Sem barra no final"
        />
        <Field
          label={integration ? "Nova API Key (deixe em branco para manter a atual)" : "API Key"}
          id="api_key"
          type="password"
          value={form.api_key}
          onChange={(v) => setForm((f) => ({ ...f, api_key: v }))}
          placeholder="••••••••••••••••"
          required={!integration}
        />
        <Field
          label="Account ID"
          id="account_id"
          value={form.account_id}
          onChange={(v) => setForm((f) => ({ ...f, account_id: v }))}
          placeholder="1"
          hint="Normalmente é 1 em instâncias self-hosted"
        />
        <Field
          label="Inbox ID (opcional)"
          id="inbox_id"
          value={form.inbox_id}
          onChange={(v) => setForm((f) => ({ ...f, inbox_id: v }))}
          placeholder="3"
          hint="ID da caixa de entrada onde as conversas serão abertas"
        />

        {error && (
          <p className="text-sm text-red-400">{error}</p>
        )}
        {saved && (
          <p className="text-sm text-green-400 flex items-center gap-1">
            <CheckCircle size={14} /> Integração salva com sucesso
          </p>
        )}

        <button
          type="submit"
          disabled={saving}
          className="self-start px-4 py-2 bg-v-accent text-white text-sm font-medium rounded-lg hover:bg-v-glow disabled:opacity-50"
        >
          {saving ? "Salvando..." : integration ? "Atualizar integração" : "Salvar integração"}
        </button>
      </form>
    </div>
  );
}
