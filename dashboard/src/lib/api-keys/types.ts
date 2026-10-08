export type ApiKeyStatus = "active" | "disable" | "cancelled";

export interface ApiKey {
  id: number;
  api_key: string;
  web_hook: string;
  user_id: number;
  status: ApiKeyStatus;
  domain_count: number;
}

export interface CreateApiKeyResponse {
  message: string; // the new api key value (show ONCE)
}

export interface ApiKeyPage {
  items: ApiKey[];
  next_cursor?: number;
  limit: number;
  has_more: boolean;
}

export type WebhookStatus = "pending" | "delivered" | "failed";

export interface WebhookLog {
  id: number;
  web_hook: string;
  event: string;
  status: WebhookStatus;
  payload: string;
  response_status: number;
  response_body: string;
  attempts: number;
  last_error: string;
  api_key_id: number;
  user_id: number;
  created_at: string;
  updated_at: string;
}

export interface WebhookLogPage {
  items: WebhookLog[];
  next_cursor?: number;
  limit: number;
  has_more: boolean;
}

export interface WebhookLogFilters {
  status?: WebhookStatus;
  event?: string;
  api_key_id?: number;
  response_status?: number;
  from?: string;
  to?: string;
  cursor?: number;
  limit?: number;
}
