import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/proxy";

export async function GET(req: NextRequest) {
  return proxyRequest(req, "/crm/integrations");
}

export async function POST(req: NextRequest) {
  return proxyRequest(req, "/crm/integrations", "POST");
}
