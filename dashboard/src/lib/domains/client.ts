import { request } from "@/lib/api/http";
import type { Domain, VerifyDomainResponse } from "./types";

export const domainsClient = {
  list: () => request<Domain[]>("/v1/dashboard/domains"),

  get: (domain: string) =>
    request<Domain>("/v1/dashboard/domain", "GET", undefined, { domain }),

  verify: (domain: string) =>
    request<VerifyDomainResponse>("/v1/dashboard/domain/verify", "POST", {
      domain,
    }),
};
