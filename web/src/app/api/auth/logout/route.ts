import { cookies } from "next/headers";
import { NextResponse } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function POST() {
  const jar = await cookies();
  const token = jar.get("access_token")?.value;

  if (token) {
    await fetch(`${API_URL}/auth/logout`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => {});
  }

  jar.delete("access_token");
  jar.delete("refresh_token");

  return NextResponse.json({ ok: true });
}
