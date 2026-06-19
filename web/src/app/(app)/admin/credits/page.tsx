"use client";

import { useEffect, useState } from "react";

interface Org {
  id: string;
  name: string;
}

type Status = "idle" | "loading" | "success" | "error";

const inputClass = "border border-v-border rounded-lg px-3 py-2 text-sm text-v-text bg-v-bg placeholder:text-v-muted focus:outline-none focus:ring-2 focus:ring-v-accent";

export default function AdminCreditsPage() {
  const [orgs, setOrgs] = useState<Org[]>([]);
  const [orgID, setOrgID] = useState("");
  const [amount, setAmount] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<Status>("idle");
  const [errorMsg, setErrorMsg] = useState("");

  useEffect(() => {
    fetch("/api/admin/organizations?limit=100")
      .then((r) => (r.ok ? r.json() : { data: [] }))
      .then((json) => {
        setOrgs(json.data ?? []);
        if (json.data?.length > 0) setOrgID(json.data[0].id);
      })
      .catch(() => setOrgs([]));
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const n = parseInt(amount, 10);
    if (!orgID || isNaN(n) || n <= 0) {
      setErrorMsg("Preencha todos os campos com valores válidos.");
      setStatus("error");
      return;
    }

    setStatus("loading");
    setErrorMsg("");

    try {
      const res = await fetch("/api/admin/credits/add", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          org_id: orgID,
          amount: n,
          description: description || `Adição manual: ${n} créditos`,
        }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        setErrorMsg(body.error ?? `Erro ${res.status}`);
        setStatus("error");
        return;
      }

      setStatus("success");
      setAmount("");
      setDescription("");
    } catch {
      setErrorMsg("Erro de rede. Tente novamente.");
      setStatus("error");
    }
  }

  return (
    <div className="max-w-lg">
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-v-text">Distribuir Créditos</h1>
        <p className="text-sm text-v-muted mt-0.5">
          Adiciona créditos ao saldo de uma organização.
        </p>
      </div>

      <form
        onSubmit={handleSubmit}
        className="bg-v-card border border-v-card-border rounded-xl p-6 flex flex-col gap-5"
      >
        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-v-text/80" htmlFor="org">
            Organização
          </label>
          <select
            id="org"
            value={orgID}
            onChange={(e) => setOrgID(e.target.value)}
            required
            className={inputClass}
          >
            {orgs.length === 0 && (
              <option value="" disabled>
                Carregando…
              </option>
            )}
            {orgs.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-v-text/80" htmlFor="amount">
            Quantidade de créditos
          </label>
          <input
            id="amount"
            type="number"
            min={1}
            step={1}
            placeholder="ex: 1000"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            required
            className={inputClass}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-v-text/80" htmlFor="desc">
            Descrição <span className="text-v-muted font-normal">(opcional)</span>
          </label>
          <input
            id="desc"
            type="text"
            placeholder="ex: Compra plano Pro"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className={inputClass}
          />
        </div>

        {status === "error" && (
          <p className="text-sm text-red-400 bg-red-900/30 border border-red-900/50 rounded-lg px-3 py-2">
            {errorMsg}
          </p>
        )}

        {status === "success" && (
          <p className="text-sm text-green-400 bg-green-900/30 border border-green-800 rounded-lg px-3 py-2">
            Créditos adicionados com sucesso.
          </p>
        )}

        <button
          type="submit"
          disabled={status === "loading"}
          className="bg-v-accent hover:bg-v-glow disabled:opacity-50 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          {status === "loading" ? "Adicionando…" : "Adicionar créditos"}
        </button>
      </form>
    </div>
  );
}
