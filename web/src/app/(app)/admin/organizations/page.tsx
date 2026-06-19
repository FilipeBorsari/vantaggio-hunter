import { cookies } from "next/headers";
import Link from "next/link";
import { Plus } from "lucide-react";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface Org {
  id: string;
  name: string;
  plan_name?: string;
  user_count: number;
  is_active: boolean;
  created_at: string;
}

async function fetchOrgs(token: string) {
  const res = await fetch(`${API_URL}/admin/organizations?limit=50`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (!res.ok) return { data: [] as Org[], total: 0 };
  return res.json() as Promise<{ data: Org[]; total: number }>;
}

export default async function OrganizationsPage() {
  const jar = await cookies();
  const token = jar.get("access_token")?.value ?? "";
  const { data: orgs, total } = await fetchOrgs(token);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold text-v-text">Organizações</h1>
          <p className="text-sm text-v-muted mt-0.5">{total} no total</p>
        </div>
        <Link
          href="/admin/organizations/new"
          className="inline-flex items-center gap-2 bg-v-accent hover:bg-v-glow text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus size={16} />
          Nova Organização
        </Link>
      </div>

      <div className="bg-v-card rounded-xl border border-v-card-border overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-v-border bg-v-bg">
              <th className="text-left font-medium text-v-muted px-4 py-3">Nome</th>
              <th className="text-left font-medium text-v-muted px-4 py-3">Plano</th>
              <th className="text-left font-medium text-v-muted px-4 py-3">Usuários</th>
              <th className="text-left font-medium text-v-muted px-4 py-3">Status</th>
              <th className="text-left font-medium text-v-muted px-4 py-3">Criado em</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-v-card-border">
            {orgs.length === 0 && (
              <tr>
                <td colSpan={5} className="text-center text-v-muted py-10">
                  Nenhuma organização encontrada
                </td>
              </tr>
            )}
            {orgs.map((org) => (
              <tr key={org.id} className="hover:bg-v-border/30 transition-colors">
                <td className="px-4 py-3 font-medium text-v-text">
                  <Link href={`/admin/organizations/${org.id}`} className="hover:text-v-accent">
                    {org.name}
                  </Link>
                </td>
                <td className="px-4 py-3 text-v-text/60">{org.plan_name ?? "—"}</td>
                <td className="px-4 py-3 text-v-text/60">{org.user_count}</td>
                <td className="px-4 py-3">
                  <span
                    className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                      org.is_active
                        ? "bg-green-900/30 text-green-400"
                        : "bg-red-900/30 text-red-400"
                    }`}
                  >
                    {org.is_active ? "Ativa" : "Inativa"}
                  </span>
                </td>
                <td className="px-4 py-3 text-v-muted">
                  {new Date(org.created_at).toLocaleDateString("pt-BR")}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
