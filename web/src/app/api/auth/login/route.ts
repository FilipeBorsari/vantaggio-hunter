import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

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

  return NextResponse.json({ ok: true });
}
