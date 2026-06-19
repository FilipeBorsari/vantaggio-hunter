import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/proxy";

export async function GET(req: NextRequest) {
  return proxyRequest(req, "/me/profile");
}

export async function PATCH(req: NextRequest) {
  return proxyRequest(req, "/me/profile");
}
