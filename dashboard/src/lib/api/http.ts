import { API_BASE_URL, MAILFORGE_BASE_URL } from "@/lib/config";

// mailforge is no longer a separate service: it was merged into the api and is
// served by it under /api (the api's own routes live under /v1), so the same
// origin and the same access token cover both. This stays a separate constant
// because the two halves keep separate path prefixes and error shapes.
// Both come from @/lib/config.

const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

export interface ApiError {
  Code: string;
  Message: string;
  StatusCode: number;

  // mailforge's error shape.
  code: string;
  message: string;
  status_code: number;
}

export class ApiClientError extends Error {
  code: string;
  statusCode: number;
  constructor(code: string, message: string, statusCode: number) {
    super(message);
    this.name = "ApiClientError";
    this.code = code;
    this.statusCode = statusCode;
  }
}

let refreshPromise: Promise<string> | null = null;

async function refreshAccessToken(): Promise<string> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = (async () => {
    try {
      // The API also sets the refresh token as a cookie, so an absent
      // localStorage token is not a failure — send the request anyway and let
      // the refresh_token cookie carry it.
      const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
      const res = await fetch(`${API_BASE_URL}/v1/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(
          refreshToken ? { refresh_token: refreshToken } : {},
        ),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        localStorage.removeItem(ACCESS_TOKEN_KEY);
        localStorage.removeItem(REFRESH_TOKEN_KEY);
        const err = data as Partial<ApiError>;
        throw new ApiClientError(
          err.Code ?? "UNAUTHORIZED_TOKEN",
          err.Message ?? "Refresh failed",
          err.StatusCode ?? res.status,
        );
      }
      const accessToken = data.access_token;
      if (typeof accessToken !== "string" || accessToken.length === 0) {
        localStorage.removeItem(ACCESS_TOKEN_KEY);
        localStorage.removeItem(REFRESH_TOKEN_KEY);
        throw new ApiClientError(
          "UNAUTHORIZED_TOKEN",
          "Invalid refresh response",
          401,
        );
      }
      localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
      if (typeof data.refresh_token === "string" && data.refresh_token.length > 0) {
        localStorage.setItem(REFRESH_TOKEN_KEY, data.refresh_token);
      }
      return accessToken;
    } finally {
      refreshPromise = null;
    }
  })();
  return refreshPromise;
}

export async function request<T>(
  path: string,
  method: string = "GET",
  body?: unknown,
  params?: Record<string, string | number | undefined>,
  isRetry: boolean = false,
): Promise<T> {
  return requestAt<T>(API_BASE_URL, path, method, body, params, isRetry);
}

/**
 * requestMail talks to mailforge (served by the api under /api). Auth, refresh and error shapes
 * are identical to request(); only the origin differs.
 */
export async function requestMail<T>(
  path: string,
  method: string = "GET",
  body?: unknown,
  params?: Record<string, string | number | undefined>,
): Promise<T> {
  return requestAt<T>(MAILFORGE_BASE_URL, path, method, body, params, false);
}

/**
 * requestMailForm posts multipart form data to mailforge — used for campaign
 * audience uploads, where the body must not be JSON-encoded.
 */
export async function requestMailForm<T>(
  path: string,
  form: FormData,
  params?: Record<string, string | number | undefined>,
): Promise<T> {
  const url = new URL(`${MAILFORGE_BASE_URL}${path}`);
  appendParams(url, params);

  const headers: Record<string, string> = {};
  const accessToken = localStorage.getItem(ACCESS_TOKEN_KEY);
  if (accessToken) headers["Authorization"] = `Bearer ${accessToken}`;
  // Content-Type is deliberately omitted so the browser sets the multipart
  // boundary itself.

  const response = await fetch(url.toString(), {
    method: "POST",
    headers,
    credentials: "include",
    body: form,
  });

  return handleResponse<T>(response);
}

function appendParams(
  url: URL,
  params?: Record<string, string | number | undefined>,
): void {
  if (!params) return;
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === "" || value === 0) return;
    url.searchParams.append(key, String(value));
  });
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (response.status === 204) return undefined as T;

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    const err = data as Partial<ApiError>;
    throw new ApiClientError(
      err.Code ?? err.code ?? "UNKNOWN",
      err.Message ?? err.message ?? "An unexpected error occurred",
      err.StatusCode ?? err.status_code ?? response.status,
    );
  }
  return data as T;
}

async function requestAt<T>(
  baseUrl: string,
  path: string,
  method: string,
  body?: unknown,
  params?: Record<string, string | number | undefined>,
  isRetry: boolean = false,
): Promise<T> {
  const url = new URL(`${baseUrl}${path}`);
  appendParams(url, params);

  const accessToken = localStorage.getItem(ACCESS_TOKEN_KEY);
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (accessToken) headers["Authorization"] = `Bearer ${accessToken}`;

  const response = await fetch(url.toString(), {
    method,
    headers,
    credentials: "include",
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (response.status === 204) return undefined as T;

  const data = await response.json().catch(() => ({}));

  if (!response.ok) {
    const err = data as Partial<ApiError>;
    // the api's own handlers send capitalised keys; mailforge sends lowercase ones.
    const code = err.Code ?? err.code ?? "UNKNOWN";
    const message = err.Message ?? err.message ?? "An unexpected error occurred";
    const statusCode = err.StatusCode ?? err.status_code ?? response.status;

    if (statusCode === 401 && code === "TOKEN_EXPIRED" && !isRetry) {
      await refreshAccessToken();
      return requestAt<T>(baseUrl, path, method, body, params, true);
    }

    throw new ApiClientError(code, message, statusCode);
  }

  return data as T;
}
