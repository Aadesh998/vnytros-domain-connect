import { request } from "@/lib/api/http";
import type {
  ApiKey,
  ApiKeyPage,
  ApiKeyStatus,
  CreateApiKeyResponse,
  WebhookLogFilters,
  WebhookLogPage,
} from "./types";

export const apiKeysClient = {
  create: (web_hook: string) =>
    request<CreateApiKeyResponse>("/v1/create/api", "POST", { web_hook }),

  list: (cursor: number = 0, limit: number = 20) =>
    request<ApiKeyPage>("/v1/get/apis", "GET", undefined, { cursor, limit }),

  update: (id: number, patch: { web_hook?: string; status?: ApiKeyStatus }) =>
    request<ApiKey>("/v1/update/api", "PATCH", patch, { id }),

  delete: (id: number) =>
    request<{ message: string }>("/v1/delete/api", "DELETE", undefined, { id }),

  filterWebhookLogs: (filters: WebhookLogFilters) =>
    request<WebhookLogPage>(
      "/v1/filter/webhooks",
      "GET",
      undefined,
      filters as Record<string, string | number | undefined>,
    ),
};
