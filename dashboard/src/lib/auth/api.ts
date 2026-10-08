import { ApiClientError, type ApiError } from "@/lib/api/http";

import { API_BASE_URL } from "@/lib/config";

const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";

interface LoginResponse {
  id: number;
  name: string;
  email: string;
  user_type: string;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

interface SignupResponse {
  message: string;
  email: string;
}

async function parseJson<T>(res: Response): Promise<T> {
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const err = data as Partial<ApiError>;
    throw new ApiClientError(
      err.Code ?? "UNKNOWN",
      err.Message ?? "Request failed",
      err.StatusCode ?? res.status,
    );
  }
  return data as T;
}

export type OAuthProvider = "google" | "github";

/**
 * Where the provider sends the browser back. This origin must be registered
 * with Google/GitHub *and* listed in the API's OAUTH_ALLOWED_CALLBACK_URLS,
 * because the API replays the same value on the token exchange.
 */
function oauthRedirectUri(provider: OAuthProvider): string {
  return `${window.location.origin}/auth/${provider}/callback`;
}

const OAUTH_NEXT_KEY = "oauth_next";

/** Park where the user was headed — the callback URL carries no query string. */
function stashOAuthNext(next: string | null | undefined): void {
  try {
    if (next && next.startsWith("/")) sessionStorage.setItem(OAUTH_NEXT_KEY, next);
    else sessionStorage.removeItem(OAUTH_NEXT_KEY);
  } catch {
    // Private-mode browsers: fall back to the default destination.
  }
}

export function takeOAuthNext(): string | null {
  try {
    const next = sessionStorage.getItem(OAUTH_NEXT_KEY);
    sessionStorage.removeItem(OAUTH_NEXT_KEY);
    return next && next.startsWith("/") ? next : null;
  } catch {
    return null;
  }
}

export const authApi = {
  async login(email: string, password: string): Promise<LoginResponse> {
    const r = await fetch(`${API_BASE_URL}/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ email, password }),
    });
    const data = await parseJson<LoginResponse>(r);
    localStorage.setItem(ACCESS_TOKEN_KEY, data.access_token);
    localStorage.setItem(REFRESH_TOKEN_KEY, data.refresh_token);
    return data;
  },

  async signup(
    name: string,
    email: string,
    password: string,
  ): Promise<SignupResponse> {
    const r = await fetch(`${API_BASE_URL}/v1/auth/signup`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ name, email, password }),
    });
    return parseJson<SignupResponse>(r);
  },

  /**
   * Confirms the email from the signup mail. The API's verification link is
   * `${FRONTEND_URL}/verify?token=…`, so point FRONTEND_URL at this dashboard.
   * Signs the user in on success.
   */
  async verify(token: string): Promise<LoginResponse> {
    const r = await fetch(
      `${API_BASE_URL}/v1/auth/verify?token=${encodeURIComponent(token)}`,
      { credentials: "include" },
    );
    const data = await parseJson<LoginResponse>(r);
    if (data.access_token) {
      localStorage.setItem(ACCESS_TOKEN_KEY, data.access_token);
      localStorage.setItem(REFRESH_TOKEN_KEY, data.refresh_token);
    }
    return data;
  },

  /** Emails a reset link (`${FRONTEND_URL}/reset-password?token=…`). */
  async forgetPassword(email: string): Promise<{ message: string }> {
    const r = await fetch(`${API_BASE_URL}/v1/auth/forget-password`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email }),
    });
    return parseJson<{ message: string }>(r);
  },

  async resetPassword(input: {
    token: string;
    new_password: string;
    confirm_password: string;
  }): Promise<{ message: string }> {
    const r = await fetch(`${API_BASE_URL}/v1/auth/reset-password`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });
    return parseJson<{ message: string }>(r);
  },

  /** Full-page redirect to the provider's consent screen. */
  startOAuth(provider: OAuthProvider, next?: string | null): void {
    stashOAuthNext(next);
    const qs = new URLSearchParams({
      redirect_uri: oauthRedirectUri(provider),
    }).toString();
    window.location.href = `${API_BASE_URL}/v1/auth/${provider}/login?${qs}`;
  },

  /** Trade the provider's authorization code for our own token pair. */
  async oauthExchange(
    provider: OAuthProvider,
    code: string,
    state: string,
  ): Promise<LoginResponse> {
    const qs = new URLSearchParams({ code, state }).toString();
    const r = await fetch(
      `${API_BASE_URL}/v1/auth/${provider}/callback?${qs}`,
      { credentials: "include" },
    );
    const data = await parseJson<LoginResponse>(r);
    localStorage.setItem(ACCESS_TOKEN_KEY, data.access_token);
    localStorage.setItem(REFRESH_TOKEN_KEY, data.refresh_token);
    return data;
  },

  async logout(): Promise<void> {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    await fetch(`${API_BASE_URL}/v1/auth/logout`, {
      method: "POST",
      credentials: "include",
    }).catch(() => {});
  },
};
