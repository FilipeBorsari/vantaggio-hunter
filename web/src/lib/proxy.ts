import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function proxyRequest(
  req: NextRequest,
  path: string,
  method?: string
): Promise<NextResponse> {
  const jar = await cookies();
  const token = jar.get("access_token")?.value;

  const init: RequestInit = {
    method: method ?? req.method,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  };

  if (req.method !== "GET" && req.method !== "HEAD") {
    init.body = await req.text();
  }

  const url = new URL(`${API_URL}${path}`);
  req.nextUrl.searchParams.forEach((v, k) => url.searchParams.set(k, v));

  const upstream = await fetch(url.toString(), init);
  const body = await upstream.text();

  return new NextResponse(body || null, {
    status: upstream.status,
    headers: { "Content-Type": "application/json" },
  });
}
