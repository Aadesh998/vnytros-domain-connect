import { useState, useEffect, useCallback } from "react";
import { userClient } from "@/lib/user/client";
import { ApiClientError } from "@/lib/api/http";
import type { User } from "@/lib/user/types";

export function useUser() {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [unauthenticated, setUnauthenticated] = useState(false);

  // State is only written from the promise callbacks, so the initial load can
  // run from an effect without a synchronous setState in the effect body.
  const load = useCallback(
    () =>
      userClient
        .getProfile()
        .then(
          (data) => {
            setUser(data);
            setUnauthenticated(false);
            setError(null);
          },
          (err: unknown) => {
            setUser(null);
            if (err instanceof ApiClientError && err.statusCode === 401) {
              setUnauthenticated(true);
              setError(null);
            } else {
              setError(err instanceof Error ? err.message : "Failed to fetch user");
            }
          },
        )
        .finally(() => setIsLoading(false)),
    [],
  );

  const refresh = useCallback(() => {
    setIsLoading(true);
    setError(null);
    return load();
  }, [load]);

  useEffect(() => {
    void load();
  }, [load]);

  return { user, isLoading, error, unauthenticated, refresh };
}
