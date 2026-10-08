import { useState, useEffect, useCallback } from "react";
import { apiKeysClient } from "@/lib/api-keys/client";
import type { ApiKey, ApiKeyStatus } from "@/lib/api-keys/types";

export function useApiKeys() {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<number | undefined>(undefined);

  // State is only written from the promise callbacks, so the initial load can
  // run from an effect without a synchronous setState in the effect body.
  const load = useCallback(
    () =>
      apiKeysClient
        .list(0)
        .then(
          (data) => {
            setKeys(data.items);
            setHasMore(data.has_more);
            setNextCursor(data.next_cursor);
            setError(null);
          },
          (err: unknown) => {
            setError(err instanceof Error ? err.message : "Failed to fetch API keys");
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

  const loadMore = useCallback(async () => {
    if (!hasMore || nextCursor === undefined || isLoading) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await apiKeysClient.list(nextCursor);
      setKeys((prev) => [...prev, ...data.items]);
      setHasMore(data.has_more);
      setNextCursor(data.next_cursor);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load more API keys");
    } finally {
      setIsLoading(false);
    }
  }, [hasMore, nextCursor, isLoading]);

  useEffect(() => {
    void load();
  }, [load]);

  const createKey = async (web_hook: string) => {
    const response = await apiKeysClient.create(web_hook);
    await refresh();
    return response.message;
  };

  const updateKey = async (
    id: number,
    patch: { web_hook?: string; status?: ApiKeyStatus },
  ) => {
    await apiKeysClient.update(id, patch);
    await refresh();
  };

  const deleteKey = async (id: number) => {
    await apiKeysClient.delete(id);
    await refresh();
  };

  return {
    keys,
    isLoading,
    error,
    hasMore,
    loadMore,
    refresh,
    createKey,
    updateKey,
    deleteKey,
  };
}
