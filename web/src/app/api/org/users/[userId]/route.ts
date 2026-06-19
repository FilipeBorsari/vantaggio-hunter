import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/proxy";

export async function PATCH(
  req: NextRequest,
  { params }: { params: Promise<{ userId: string }> }
) {
  const { userId } = await params;
  return proxyRequest(req, `/org/users/${userId}`);
}

export async function DELETE(
  req: NextRequest,
  { params }: { params: Promise<{ userId: string }> }
) {
  const { userId } = await params;
  return proxyRequest(req, `/org/users/${userId}`);
}
