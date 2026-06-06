"use client";

import { createApiClient } from "@/lib/api/client";
import type { AuthResponse, User } from "@/lib/api/types";

const accessKey = "netquest.accessToken";
const refreshKey = "netquest.refreshToken";
const userKey = "netquest.user";

export function getAccessToken() {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(accessKey);
}

export function getRefreshToken() {
  return null;
}

export function getStoredUser(): User | null {
  if (typeof window === "undefined") return null;
  const raw = window.localStorage.getItem(userKey);
  return raw ? (JSON.parse(raw) as User) : null;
}

export function storeAuth(response: AuthResponse) {
  window.localStorage.setItem(accessKey, response.accessToken);
  window.localStorage.removeItem(refreshKey);
  window.localStorage.setItem(userKey, JSON.stringify(response.user));
}

export function clearAuth() {
  window.localStorage.removeItem(accessKey);
  window.localStorage.removeItem(refreshKey);
  window.localStorage.removeItem(userKey);
}

export function authedClient() {
  return createApiClient({ getToken: getAccessToken });
}
