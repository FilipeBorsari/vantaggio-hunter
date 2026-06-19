"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  Brain,
  Building2,
  CreditCard,
  DollarSign,
  LayoutDashboard,
  LogOut,
  Menu,
  Search,
  Settings,
  ShieldCheck,
  Upload,
  Users,
  X,
} from "lucide-react";
import CreditBalance from "@/components/credits/CreditBalance";
import { useState } from "react";
import { UserRole } from "@/lib/auth";

interface AppLayoutProps {
  children: React.ReactNode;
  role: UserRole;
  userEmail?: string;
}

const navItems = [
  // super_admin
  {
    href: "/admin",
    label: "Dashboard Global",
    icon: ShieldCheck,
    roles: ["super_admin"] as string[],
    exact: true,
  },
  {
    href: "/admin/organizations",
    label: "Organizações",
    icon: Building2,
    roles: ["super_admin"] as string[],
  },
  // org_admin
  {
    href: "/org",
    label: "Dashboard",
    icon: LayoutDashboard,
    roles: ["org_admin"] as string[],
    exact: true,
  },
  {
    href: "/org/users",
    label: "Vendedores",
    icon: Users,
    roles: ["org_admin"] as string[],
  },
  {
    href: "/org/costs",
    label: "Custos",
    icon: DollarSign,
    roles: ["org_admin"] as string[],
  },
  {
    href: "/org/credits",
    label: "Créditos",
    icon: CreditCard,
    roles: ["org_admin"] as string[],
  },
  // seller
  {
    href: "/dashboard",
    label: "Dashboard",
    icon: LayoutDashboard,
    roles: ["seller"] as string[],
  },
  {
    href: "/search",
    label: "Buscar Leads",
    icon: Search,
    roles: ["seller"] as string[],
  },
  {
    href: "/companies",
    label: "Lead Bank",
    icon: Building2,
    roles: ["seller"] as string[],
  },
  {
    href: "/intelligence",
    label: "Inteligência",
    icon: Brain,
    roles: ["seller"] as string[],
  },
  {
    href: "/exports",
    label: "Exportações",
    icon: Upload,
    roles: ["seller"] as string[],
  },
  {
    href: "/me/searches",
    label: "Minhas Pesquisas",
    icon: Search,
    roles: ["seller"] as string[],
  },
  {
    href: "/settings/crm",
    label: "Config. CRM",
    icon: Settings,
    roles: ["seller"] as string[],
  },
];

interface SidebarContentProps {
  visibleItems: (typeof navItems)[number][];
  pathname: string;
  role: UserRole;
  onNavigate: () => void;
  onLogout: () => void;
}

function SidebarContent({ visibleItems, pathname, onNavigate, onLogout }: SidebarContentProps) {
  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-5 border-b border-v-card-border">
        <span className="text-base font-bold text-v-accent">Vantaggio Hunter</span>
      </div>
      <nav className="flex-1 px-3 py-4 space-y-1">
        {visibleItems.map((item) => {
          const Icon = item.icon;
          const active = item.exact ? pathname === item.href : pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={onNavigate}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                active
                  ? "bg-v-accent/10 text-v-accent"
                  : "text-v-muted hover:bg-v-border/40 hover:text-v-text"
              }`}
            >
              <Icon size={18} />
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="px-3 py-4 border-t border-v-card-border space-y-1">
        <Link
          href="/profile"
          onClick={onNavigate}
          className="flex items-center gap-3 px-3 py-2 text-sm text-v-muted hover:bg-v-border/40 hover:text-v-text rounded-lg transition-colors"
        >
          Perfil
        </Link>
        <button
          onClick={onLogout}
          className="flex items-center gap-3 w-full px-3 py-2 text-sm text-v-muted hover:bg-v-border/40 hover:text-v-text rounded-lg transition-colors"
        >
          <LogOut size={18} />
          Sair
        </button>
      </div>
    </div>
  );
}

export default function AppLayout({ children, role, userEmail }: AppLayoutProps) {
  const pathname = usePathname();
  const router = useRouter();
  const [mobileOpen, setMobileOpen] = useState(false);

  const visibleItems = navItems.filter((item) => item.roles.includes(role));

  async function handleLogout() {
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/login");
    router.refresh();
  }

  return (
    <div className="flex h-screen bg-v-bg">
      {/* Sidebar desktop */}
      <aside className="hidden md:flex flex-col w-56 bg-v-card border-r border-v-border shrink-0">
        <SidebarContent
          visibleItems={visibleItems}
          pathname={pathname}
          role={role}
          onNavigate={() => setMobileOpen(false)}
          onLogout={handleLogout}
        />
      </aside>

      {/* Mobile sidebar */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 md:hidden">
          <div
            className="absolute inset-0 bg-black/60"
            onClick={() => setMobileOpen(false)}
          />
          <aside className="relative w-56 h-full bg-v-card border-r border-v-border shadow-xl">
            <SidebarContent
              visibleItems={visibleItems}
              pathname={pathname}
              role={role}
              onNavigate={() => setMobileOpen(false)}
              onLogout={handleLogout}
            />
          </aside>
        </div>
      )}

      {/* Main content */}
      <div className="flex flex-col flex-1 min-w-0">
        {/* Topbar */}
        <header className="flex items-center justify-between h-14 px-4 bg-v-card border-b border-v-border shrink-0">
          <button
            className="md:hidden p-1 rounded-md text-v-muted hover:text-v-text"
            onClick={() => setMobileOpen(true)}
          >
            <Menu size={20} />
          </button>
          <div className="flex items-center gap-4 ml-auto">
            {role !== "super_admin" && <CreditBalance />}
            <div className="flex items-center gap-2">
              <span className="text-xs text-v-muted uppercase tracking-wide">{role.replace("_", " ")}</span>
              <span className="text-sm text-v-text/70">{userEmail}</span>
            </div>
            <button
              onClick={handleLogout}
              className="ml-2 p-1 rounded-md text-v-muted hover:text-v-text md:hidden"
              title="Sair"
            >
              <X size={18} />
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </div>
    </div>
  );
}
