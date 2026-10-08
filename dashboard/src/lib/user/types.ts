export type UserType = "user" | "admin";

export interface User {
  id: number;
  email: string;
  user_type: UserType;
  name?: string;
  country?: string;
  city?: string;
  verified?: boolean;
  domain_count?: number;
}
