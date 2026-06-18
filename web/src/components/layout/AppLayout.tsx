"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  Brain,
  Building2,
  CreditCard,
  LayoutDashboard,
  LogOut,
  Menu,
  Search,
  ShieldCheck,
  X,
} from "lucide-react";
import CreditBalance from "@/components/credits/CreditBalance";
import { useState } from "react";

interface AppLayoutProps {
  children: React.ReactNode;
  role: "admin" | "manager" | "operator";
  userEmail?: string;
}

const navItems = [
  {
    href: "/search",
    label: "Busca",
    icon: Search,
    roles: ["admin", "manager", "operator"] as string[],
  },
  {
    href: "/companies",
    label: "Lead Bank",
    icon: Building2,
    roles: ["admin", "manager", "operator"] as string[],
  },
  {
    href: "/intelligence",
    label: "Inteligência",
    icon: Brain,
    roles: ["admin", "manager", "operator"] as string[],
  },
  {
    href: "/credits",
    label: "Créditos",
    icon: CreditCard,
    roles: ["admin", "manager"] as string[],
  },
  {
    href: "/admin/organizations",
    label: "Organizações",
    icon: ShieldCheck,
    roles: ["admin"] as string[],
  },
  {
    href: "/admin/credits",
    label: "Distribuir Créditos",
    icon: CreditCard,
    roles: ["admin"] as string[],
  },
];

interface SidebarContentProps {
  visibleItems: (typeof navItems)[number][];
  pathname: string;
  onNavigate: () => void;
  onLogout: () => void;
}

function SidebarContent({ visibleItems, pathname, onNavigate, onLogout }: SidebarContentProps) {
  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-5 border-b border-gray-100">
        <span className="text-base font-bold text-indigo-600">Vantaggio Hunter</span>
      </div>
      <nav className="flex-1 px-3 py-4 space-y-1">
        {visibleItems.map((item) => {
          const Icon = item.icon;
          const active = pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={onNavigate}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                active
                  ? "bg-indigo-50 text-indigo-700"
                  : "text-gray-600 hover:bg-gray-100 hover:text-gray-900"
              }`}
            >
              <Icon size={18} />
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="px-3 py-4 border-t border-gray-100">
        <button
          onClick={onLogout}
          className="flex items-center gap-3 w-full px-3 py-2 text-sm text-gray-600 hover:bg-gray-100 hover:text-gray-900 rounded-lg transition-colors"
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
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar desktop */}
      <aside className="hidden md:flex flex-col w-56 bg-white border-r border-gray-200 shrink-0">
        <SidebarContent
          visibleItems={visibleItems}
          pathname={pathname}
          onNavigate={() => setMobileOpen(false)}
          onLogout={handleLogout}
        />
      </aside>

      {/* Mobile sidebar */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 md:hidden">
          <div
            className="absolute inset-0 bg-black/30"
            onClick={() => setMobileOpen(false)}
          />
          <aside className="relative w-56 h-full bg-white shadow-xl">
            <SidebarContent
              visibleItems={visibleItems}
              pathname={pathname}
              onNavigate={() => setMobileOpen(false)}
              onLogout={handleLogout}
            />
          </aside>
        </div>
      )}

      {/* Main content */}
      <div className="flex flex-col flex-1 min-w-0">
        {/* Topbar */}
        <header className="flex items-center justify-between h-14 px-4 bg-white border-b border-gray-200 shrink-0">
          <button
            className="md:hidden p-1 rounded-md text-gray-500 hover:text-gray-900"
            onClick={() => setMobileOpen(true)}
          >
            <Menu size={20} />
          </button>
          <div className="flex items-center gap-4 ml-auto">
            <CreditBalance />
            <div className="flex items-center gap-2">
              <LayoutDashboard size={16} className="text-gray-400 hidden md:block" />
              <span className="text-sm text-gray-600">{userEmail}</span>
            </div>
            <button
              onClick={handleLogout}
              className="ml-2 p-1 rounded-md text-gray-400 hover:text-gray-600 md:hidden"
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
