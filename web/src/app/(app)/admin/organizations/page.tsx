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
          <h1 className="text-xl font-semibold text-gray-900">Organizações</h1>
          <p className="text-sm text-gray-500 mt-0.5">{total} no total</p>
        </div>
        <Link
          href="/admin/organizations/new"
          className="inline-flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus size={16} />
          Nova Organização
        </Link>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-100 bg-gray-50">
              <th className="text-left font-medium text-gray-600 px-4 py-3">Nome</th>
              <th className="text-left font-medium text-gray-600 px-4 py-3">Plano</th>
              <th className="text-left font-medium text-gray-600 px-4 py-3">Usuários</th>
              <th className="text-left font-medium text-gray-600 px-4 py-3">Status</th>
              <th className="text-left font-medium text-gray-600 px-4 py-3">Criado em</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {orgs.length === 0 && (
              <tr>
                <td colSpan={5} className="text-center text-gray-400 py-10">
                  Nenhuma organização encontrada
                </td>
              </tr>
            )}
            {orgs.map((org) => (
              <tr key={org.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-4 py-3 font-medium text-gray-900">{org.name}</td>
                <td className="px-4 py-3 text-gray-600">{org.plan_name ?? "—"}</td>
                <td className="px-4 py-3 text-gray-600">{org.user_count}</td>
                <td className="px-4 py-3">
                  <span
                    className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                      org.is_active
                        ? "bg-green-50 text-green-700"
                        : "bg-red-50 text-red-700"
                    }`}
                  >
                    {org.is_active ? "Ativa" : "Inativa"}
                  </span>
                </td>
                <td className="px-4 py-3 text-gray-500">
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
