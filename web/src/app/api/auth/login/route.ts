import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

function decodeRole(token: string): string {
  try {
    const payload = JSON.parse(Buffer.from(token.split(".")[1], "base64url").toString("utf-8"));
    return payload.role ?? "";
  } catch {
    return "";
  }
}

const ROLE_HOME: Record<string, string> = {
  super_admin: "/admin",
  org_admin: "/org",
  seller: "/companies",
};

export async function POST(req: NextRequest) {
  const body = await req.json();

  const upstream = await fetch(`${API_URL}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  if (!upstream.ok) {
    const err = await upstream.json().catch(() => ({ error: "credenciais inválidas" }));
    return NextResponse.json(err, { status: upstream.status });
  }

  const pair = await upstream.json();
  const jar = await cookies();

  jar.set("access_token", pair.access_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    maxAge: pair.expires_in,
    path: "/",
  });
  jar.set("refresh_token", pair.refresh_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    maxAge: 30 * 24 * 60 * 60,
    path: "/",
  });

  const role = decodeRole(pair.access_token);
  const redirect = ROLE_HOME[role] ?? "/companies";

  return NextResponse.json({ ok: true, redirect });
}
