import { NextRequest } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function GET(
  _req: NextRequest,
  { params }: { params: Promise<{ token: string }> }
) {
  const { token } = await params;
  const res = await fetch(`${API_URL}/invitations/${token}`);
  const body = await res.text();
  return new Response(body || null, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
