import { useState, useEffect, useCallback } from "react";
import { apiKeysClient } from "@/lib/api-keys/client";
import type { WebhookLog, WebhookLogFilters } from "@/lib/api-keys/types";

export function useWebhookLogs(initialFilters: WebhookLogFilters = {}) {
  const [logs, setLogs] = useState<WebhookLog[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState<number | undefined>(undefined);
  const [filters, setFilters] = useState<WebhookLogFilters>(initialFilters);

  // State is only written from the promise callbacks, so the load can run
  // from an effect without a synchronous setState in the effect body.
  const load = useCallback(
    () =>
      apiKeysClient
        .filterWebhookLogs({ ...filters, cursor: 0 })
        .then(
          (data) => {
            setLogs(data.items);
            setHasMore(data.has_more);
            setNextCursor(data.next_cursor);
            setError(null);
          },
          (err: unknown) => {
            setError(err instanceof Error ? err.message : "Failed to fetch webhook logs");
          },
        )
        .finally(() => setIsLoading(false)),
    [filters],
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
      const data = await apiKeysClient.filterWebhookLogs({
        ...filters,
        cursor: nextCursor,
      });
      setLogs((prev) => [...prev, ...data.items]);
      setHasMore(data.has_more);
      setNextCursor(data.next_cursor);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load more logs");
    } finally {
      setIsLoading(false);
    }
  }, [filters, hasMore, nextCursor, isLoading]);

  useEffect(() => {
    void load();
  }, [load]);

  // Changing filters starts a fresh load, so mark it loading here in the
  // event handler; the effect above picks up the new filters.
  const applyFilters = useCallback((newFilters: WebhookLogFilters) => {
    setFilters(newFilters);
    setIsLoading(true);
    setError(null);
  }, []);

  return {
    logs,
    isLoading,
    error,
    hasMore,
    loadMore,
    applyFilters,
    filters,
    refresh,
  };
}
