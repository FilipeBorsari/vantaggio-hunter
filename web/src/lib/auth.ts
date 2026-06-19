import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { apiFetch } from "./api";

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export type UserRole = "super_admin" | "org_admin" | "seller";

export interface AuthUser {
  user_id: string;
  org_id: string;
  role: UserRole;
  impersonated_by?: string;
}

export async function login(email: string, password: string): Promise<void> {
  const pair = await apiFetch<TokenPair>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });

  const jar = await cookies();
  jar.set("access_token", pair.access_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    maxAge: pair.expires_in,
    path: "/",
  });
  jar.set("refresh_token", pair.refresh_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    maxAge: 30 * 24 * 60 * 60,
    path: "/",
  });
}

export async function logout(): Promise<void> {
  const jar = await cookies();
  const token = jar.get("access_token")?.value;
  if (token) {
    await apiFetch("/auth/logout", {
      method: "POST",
      token,
    }).catch(() => {});
  }
  jar.delete("access_token");
  jar.delete("refresh_token");
  redirect("/login");
}

export async function getToken(): Promise<string | undefined> {
  const jar = await cookies();
  return jar.get("access_token")?.value;
}

export function decodeJWT(token: string): AuthUser | null {
  try {
    const payload = token.split(".")[1];
    const decoded = JSON.parse(
      Buffer.from(payload, "base64url").toString("utf-8")
    );
    return {
      user_id: decoded.user_id,
      org_id: decoded.org_id,
      role: decoded.role as UserRole,
      impersonated_by: decoded.impersonated_by,
    };
  } catch {
    return null;
  }
}
