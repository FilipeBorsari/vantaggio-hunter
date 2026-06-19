"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

interface OrgUser {
  user_id: string;
  name: string;
  email: string;
  role: string;
  is_active: boolean;
  credit_limit: number | null;
  searches_this_month: number;
  exports_this_month: number;
  credits_consumed: number;
}

interface Invitation {
  invitation_id: string;
  email: string;
  role: string;
  expires_at: string;
  accepted_at: string | null;
}

const modalInputClass = "w-full border border-v-border rounded-md px-3 py-2 text-sm text-v-text bg-v-bg focus:outline-none focus:ring-2 focus:ring-v-accent";

export default function OrgUsersPage() {
  const [users, setUsers] = useState<OrgUser[]>([]);
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [editUser, setEditUser] = useState<OrgUser | null>(null);
  const [editLimit, setEditLimit] = useState("");

  const load = () => {
    fetch("/api/org/users").then((r) => r.json()).then(setUsers);
    fetch("/api/org/invitations").then((r) => r.json()).then(setInvitations);
  };

  useEffect(load, []);

  const pending = invitations.filter((i) => !i.accepted_at);

  async function handleInvite() {
    await fetch("/api/org/invitations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: inviteEmail, role: "seller" }),
    });
    setInviteOpen(false);
    setInviteEmail("");
    load();
  }

  async function handleRevoke(id: string) {
    await fetch(`/api/org/invitations/${id}`, { method: "DELETE" });
    load();
  }

  async function handleDeactivate(userId: string) {
    await fetch(`/api/org/users/${userId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ is_active: false }),
    });
    load();
  }

  async function handleSaveLimit() {
    if (!editUser) return;
    const creditLimit = editLimit === "" ? null : parseInt(editLimit);
    await fetch(`/api/org/users/${editUser.user_id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ credit_limit: creditLimit }),
    });
    setEditUser(null);
    load();
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-v-text">Vendedores</h1>
        <button
          onClick={() => setInviteOpen(true)}
          className="px-3 py-1.5 text-sm bg-v-accent text-white rounded-md hover:bg-v-glow"
        >
          Convidar Vendedor
        </button>
      </div>

      {/* Active users */}
      <div className="bg-v-card rounded-xl border border-v-card-border overflow-hidden">
        <div className="px-4 py-3 border-b border-v-border">
          <h2 className="text-sm font-semibold text-v-text/80">Usuários Ativos</h2>
        </div>
        <table className="w-full text-sm">
          <thead className="bg-v-bg text-v-muted text-xs uppercase">
            <tr>
              <th className="px-4 py-3 text-left">Nome</th>
              <th className="px-4 py-3 text-right">Buscas/mês</th>
              <th className="px-4 py-3 text-right">Créditos</th>
              <th className="px-4 py-3 text-right">Limite</th>
              <th className="px-4 py-3 text-center">Status</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody className="divide-y divide-v-card-border">
            {users.map((u) => (
              <tr key={u.user_id} className="hover:bg-v-border/30">
                <td className="px-4 py-3">
                  <div className="font-medium text-v-text">{u.name}</div>
                  <div className="text-xs text-v-muted">{u.email}</div>
                </td>
                <td className="px-4 py-3 text-right text-v-text/60">{u.searches_this_month}</td>
                <td className="px-4 py-3 text-right text-v-text/60">{u.credits_consumed}</td>
                <td className="px-4 py-3 text-right text-v-text/60">{u.credit_limit ?? "—"}</td>
                <td className="px-4 py-3 text-center">
                  <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${u.is_active ? "bg-green-900/30 text-green-400" : "bg-v-border text-v-muted"}`}>
                    {u.is_active ? "Ativo" : "Inativo"}
                  </span>
                </td>
                <td className="px-4 py-3 text-right space-x-2">
                  <Link href={`/org/users/${u.user_id}`} className="text-xs text-v-accent hover:underline">
                    Histórico
                  </Link>
                  <button
                    onClick={() => { setEditUser(u); setEditLimit(u.credit_limit?.toString() ?? ""); }}
                    className="text-xs text-v-muted hover:text-v-text hover:underline"
                  >
                    Editar
                  </button>
                  {u.is_active && (
                    <button onClick={() => handleDeactivate(u.user_id)} className="text-xs text-red-400 hover:underline">
                      Desativar
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pending invitations */}
      {pending.length > 0 && (
        <div className="bg-v-card rounded-xl border border-v-card-border overflow-hidden">
          <div className="px-4 py-3 border-b border-v-border">
            <h2 className="text-sm font-semibold text-v-text/80">Convites Pendentes</h2>
          </div>
          <table className="w-full text-sm">
            <thead className="bg-v-bg text-v-muted text-xs uppercase">
              <tr>
                <th className="px-4 py-3 text-left">E-mail</th>
                <th className="px-4 py-3 text-left">Role</th>
                <th className="px-4 py-3 text-left">Expira em</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-v-card-border">
              {pending.map((inv) => (
                <tr key={inv.invitation_id} className="hover:bg-v-border/30">
                  <td className="px-4 py-3 text-v-text/80">{inv.email}</td>
                  <td className="px-4 py-3 text-v-text/60">{inv.role}</td>
                  <td className="px-4 py-3 text-v-muted text-xs">{new Date(inv.expires_at).toLocaleDateString("pt-BR")}</td>
                  <td className="px-4 py-3 text-right">
                    <button onClick={() => handleRevoke(inv.invitation_id)} className="text-xs text-red-400 hover:underline">
                      Revogar
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Invite Modal */}
      {inviteOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="bg-v-card border border-v-border rounded-xl shadow-lg p-6 w-full max-w-sm space-y-4">
            <h2 className="text-base font-semibold text-v-text">Convidar Vendedor</h2>
            <input
              type="email"
              placeholder="E-mail do vendedor"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
              className={modalInputClass}
            />
            <div className="flex justify-end gap-2">
              <button onClick={() => setInviteOpen(false)} className="px-3 py-1.5 text-sm text-v-muted hover:bg-v-border/40 rounded-md">
                Cancelar
              </button>
              <button onClick={handleInvite} className="px-3 py-1.5 text-sm bg-v-accent text-white rounded-md hover:bg-v-glow">
                Enviar Convite
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit credit limit modal */}
      {editUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="bg-v-card border border-v-border rounded-xl shadow-lg p-6 w-full max-w-sm space-y-4">
            <h2 className="text-base font-semibold text-v-text">Editar {editUser.name}</h2>
            <div>
              <label className="block text-xs text-v-muted mb-1">Limite de Créditos (vazio = sem limite)</label>
              <input
                type="number"
                value={editLimit}
                onChange={(e) => setEditLimit(e.target.value)}
                className={modalInputClass}
              />
            </div>
            <div className="flex justify-end gap-2">
              <button onClick={() => setEditUser(null)} className="px-3 py-1.5 text-sm text-v-muted hover:bg-v-border/40 rounded-md">
                Cancelar
              </button>
              <button onClick={handleSaveLimit} className="px-3 py-1.5 text-sm bg-v-accent text-white rounded-md hover:bg-v-glow">
                Salvar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
