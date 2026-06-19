import { NextRequest } from "next/server";
import { proxyRequest } from "@/lib/proxy";

export async function DELETE(
  req: NextRequest,
  { params }: { params: Promise<{ invitationId: string }> }
) {
  const { invitationId } = await params;
  return proxyRequest(req, `/org/invitations/${invitationId}`);
}
