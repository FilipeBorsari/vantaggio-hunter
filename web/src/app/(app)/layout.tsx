import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import AppLayout from "@/components/layout/AppLayout";
import { decodeJWT } from "@/lib/auth";

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
    <AppLayout role={claims.role} userEmail={claims.user_id}>
      {children}
    </AppLayout>
  );
}
