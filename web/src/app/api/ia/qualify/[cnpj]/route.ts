import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/proxy";

export async function POST(req: NextRequest, { params }: { params: Promise<{ cnpj: string }> }) {
  const { cnpj } = await params;
  return proxyRequest(req, `/ia/qualify/${cnpj}`, "POST");
}
