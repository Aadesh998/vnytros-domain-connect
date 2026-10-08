import { requestMail, requestMailForm } from "@/lib/api/http";
import type {
  AccountAnalytics,
  Branding,
  BrandingPayload,
  Campaign,
  CampaignAnalytics,
  CampaignPage,
  CampaignPayload,
  MailTemplate,
  RecipientList,
  RecipientStatus,
  SendCampaignResult,
  SmtpConfig,
  SmtpConfigList,
  SmtpConfigPayload,
  TemplatePage,
  TemplatePayload,
} from "./types";

/** Campaigns on mailforge (served by the api under /api). */
export const campaignsClient = {
  list: (lastId?: number, limit = 20) =>
    requestMail<CampaignPage>("/api/campaign", "GET", undefined, {
      last_id: lastId,
      limit,
    }),

  listDrafts: (lastId?: number, limit = 20) =>
    requestMail<CampaignPage>("/api/campaign/draft", "GET", undefined, {
      last_id: lastId,
      limit,
    }),

  get: (id: number) => requestMail<Campaign>(`/api/campaign/${id}`),

  create: (payload: CampaignPayload) =>
    requestMail<Campaign>("/api/campaign", "POST", payload),

  update: (id: number, payload: CampaignPayload) =>
    requestMail<Campaign>(`/api/campaign/${id}`, "PUT", payload),

  delete: (id: number) =>
    requestMail<{ success: boolean }>(`/api/campaign/${id}`, "DELETE"),

  /**
   * Queues a campaign. Resolves once the batches are accepted, not once the
   * mail is delivered — poll analytics for progress.
   */
  send: (id: number, audience: File, smtpConfigId?: number) => {
    const form = new FormData();
    form.append("file", audience);
    return requestMailForm<SendCampaignResult>(
      `/api/campaign/${id}/send`,
      form,
      { smtp_config_id: smtpConfigId },
    );
  },

  analytics: (id: number, days = 30, interval?: "hour" | "day") =>
    requestMail<CampaignAnalytics>(
      `/api/campaign/${id}/analytics`,
      "GET",
      undefined,
      { days, interval },
    ),

  recipients: (
    id: number,
    opts: { status?: RecipientStatus; offset?: number; limit?: number } = {},
  ) =>
    requestMail<RecipientList>(
      `/api/campaign/${id}/recipients`,
      "GET",
      undefined,
      { status: opts.status, offset: opts.offset, limit: opts.limit ?? 50 },
    ),
};

/** Reusable email templates. */
export const mailTemplatesClient = {
  list: (lastId?: number, limit = 20) =>
    requestMail<TemplatePage>("/api/template", "GET", undefined, {
      last_id: lastId,
      limit,
    }),

  listDrafts: (lastId?: number, limit = 20) =>
    requestMail<TemplatePage>("/api/template/draft", "GET", undefined, {
      last_id: lastId,
      limit,
    }),

  get: (id: number) => requestMail<MailTemplate>(`/api/template/${id}`),

  create: (payload: TemplatePayload) =>
    requestMail<MailTemplate>("/api/template", "POST", payload),

  update: (id: number, payload: TemplatePayload) =>
    requestMail<MailTemplate>(`/api/template/${id}`, "PUT", payload),

  delete: (id: number) =>
    requestMail<{ success: boolean }>(`/api/template/${id}`, "DELETE"),
};

/** Sender credentials — a user may keep several. */
export const smtpConfigsClient = {
  list: () => requestMail<SmtpConfigList>("/api/settings/smtp"),

  get: (id: number) => requestMail<SmtpConfig>(`/api/settings/smtp/${id}`),

  create: (payload: SmtpConfigPayload) =>
    requestMail<SmtpConfig>("/api/settings/smtp", "POST", payload),

  update: (id: number, payload: SmtpConfigPayload) =>
    requestMail<SmtpConfig>(`/api/settings/smtp/${id}`, "PUT", payload),

  setDefault: (id: number) =>
    requestMail<SmtpConfigList>(`/api/settings/smtp/${id}/default`, "POST"),

  delete: (id: number) =>
    requestMail<{ success: boolean }>(`/api/settings/smtp/${id}`, "DELETE"),

  /** Imports the service-wide fallback credentials as a first sender. */
  seed: () => requestMail<SmtpConfig>("/api/settings/smtp/seed", "POST"),
};

/** Email watermark shown in campaign footers. */
export const brandingClient = {
  get: () => requestMail<Branding>("/api/settings/branding"),
  update: (payload: BrandingPayload) =>
    requestMail<Branding>("/api/settings/branding", "PUT", payload),
};

/** Account-wide mail analytics. */
export const mailAnalyticsClient = {
  overview: (days = 30, interval?: "hour" | "day") =>
    requestMail<AccountAnalytics>("/api/analytics/overview", "GET", undefined, {
      days,
      interval,
    }),
};
