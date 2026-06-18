"use client";

import { useEffect, useState } from "react";

interface Org {
  id: string;
  name: string;
}

type Status = "idle" | "loading" | "success" | "error";

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
        <h1 className="text-xl font-semibold text-gray-900">Distribuir Créditos</h1>
        <p className="text-sm text-gray-500 mt-0.5">
          Adiciona créditos ao saldo de uma organização.
        </p>
      </div>

      <form
        onSubmit={handleSubmit}
        className="bg-white border border-gray-200 rounded-xl p-6 flex flex-col gap-5"
      >
        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-gray-700" htmlFor="org">
            Organização
          </label>
          <select
            id="org"
            value={orgID}
            onChange={(e) => setOrgID(e.target.value)}
            required
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
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
          <label className="text-sm font-medium text-gray-700" htmlFor="amount">
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
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-gray-700" htmlFor="desc">
            Descrição <span className="text-gray-400 font-normal">(opcional)</span>
          </label>
          <input
            id="desc"
            type="text"
            placeholder="ex: Compra plano Pro"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>

        {status === "error" && (
          <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
            {errorMsg}
          </p>
        )}

        {status === "success" && (
          <p className="text-sm text-green-700 bg-green-50 border border-green-200 rounded-lg px-3 py-2">
            Créditos adicionados com sucesso.
          </p>
        )}

        <button
          type="submit"
          disabled={status === "loading"}
          className="bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          {status === "loading" ? "Adicionando…" : "Adicionar créditos"}
        </button>
      </form>
    </div>
  );
}
