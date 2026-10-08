import { request } from "@/lib/api/http";
import type { User } from "./types";

export const userClient = {
  getProfile: () => request<User>("/v1/me"),
};
