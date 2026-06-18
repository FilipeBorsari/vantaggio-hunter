"use client";

import { useEffect, useState } from "react";
import { AlertTriangle, CreditCard } from "lucide-react";
import Link from "next/link";

interface BalanceResponse {
  balance: number;
  org_id: string;
}

export default function CreditBalance() {
  const [balance, setBalance] = useState<number | null>(null);

  useEffect(() => {
    let active = true;

    async function fetchBalance() {
      try {
        const res = await fetch("/api/credits/balance");
        if (res.ok) {
          const data: BalanceResponse = await res.json();
          if (active) setBalance(data.balance);
        }
      } catch {
        // silently ignore — balance is optional UI chrome
      }
    }

    fetchBalance();
    const timer = setInterval(fetchBalance, 30_000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, []);

  if (balance === null) return null;

  const colorClass =
    balance === 0
      ? "text-red-600"
      : balance < 100
      ? "text-orange-500"
      : "text-gray-600";

  return (
    <Link
      href="/credits"
      className={`flex items-center gap-1.5 text-sm font-medium hover:opacity-80 transition-opacity ${colorClass}`}
      title="Ver créditos"
    >
      {balance < 100 ? (
        <AlertTriangle size={14} />
      ) : (
        <CreditCard size={14} />
      )}
      <span>{balance.toLocaleString("pt-BR")} créditos</span>
    </Link>
  );
}
