import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import AppLayout from "@/components/layout/AppLayout";

function decodeJWT(token: string) {
  try {
    const payload = token.split(".")[1];
    return JSON.parse(Buffer.from(payload, "base64url").toString("utf-8"));
  } catch {
    return null;
  }
}

export default async function AppRootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const jar = await cookies();
  const token = jar.get("access_token")?.value;
  if (!token) redirect("/login");

  const claims = decodeJWT(token);
  if (!claims) redirect("/login");

  return (
    <AppLayout role={claims.role} userEmail={claims.email ?? claims.user_id}>
      {children}
    </AppLayout>
  );
}
